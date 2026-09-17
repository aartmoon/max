package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tvoydom/gar-init/internal/importer"
	"tvoydom/gar-init/internal/model"
)

func TestImportFileRollsBackRowsAndStateTogether(t *testing.T) {
	store := testStore(t)
	family := familyByKey(t, "houses")
	source := importer.SourceFile{Name: "AS_HOUSES_test.XML", Size: 123, Family: family}
	errBoom := errors.New("parser failed")
	_, _, err := store.ImportFile(context.Background(), source, func(ctx context.Context, write importer.BatchWriter) (int64, error) {
		if err := write(ctx, family, [][]any{emptyRow(family)}); err != nil {
			return 0, err
		}
		return 1, errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("got %v", err)
	}
	assertCount(t, store.pool, "gar_houses", 0)
	assertCount(t, store.pool, "gar_import_state", 0)
}

func TestImportFileCommitsAndSkipsCompletedFile(t *testing.T) {
	store := testStore(t)
	family := familyByKey(t, "houses")
	source := importer.SourceFile{Name: "AS_HOUSES_test.XML", Size: 123, Family: family}
	calls := 0
	parse := func(ctx context.Context, write importer.BatchWriter) (int64, error) {
		calls++
		row := emptyRow(family)
		row[columnIndex(family, "object_id")] = int64(10)
		if err := write(ctx, family, [][]any{row}); err != nil {
			return 0, err
		}
		return 1, nil
	}
	count, skipped, err := store.ImportFile(context.Background(), source, parse)
	if err != nil || skipped || count != 1 {
		t.Fatalf("count=%d skipped=%v err=%v", count, skipped, err)
	}
	count, skipped, err = store.ImportFile(context.Background(), source, parse)
	if err != nil || !skipped || count != 1 || calls != 1 {
		t.Fatalf("second import count=%d skipped=%v calls=%d err=%v", count, skipped, calls, err)
	}
	assertCount(t, store.pool, "gar_houses", 1)
	assertCount(t, store.pool, "gar_import_state", 1)
}

func TestImportFileRejectsChangedCompletedFile(t *testing.T) {
	store := testStore(t)
	family := familyByKey(t, "houses")
	source := importer.SourceFile{Name: "AS_HOUSES_test.XML", Size: 123, Family: family}
	if _, _, err := store.ImportFile(context.Background(), source, func(context.Context, importer.BatchWriter) (int64, error) { return 0, nil }); err != nil {
		t.Fatal(err)
	}
	source.Size = 124
	_, _, err := store.ImportFile(context.Background(), source, func(context.Context, importer.BatchWriter) (int64, error) {
		t.Fatal("parser must not run")
		return 0, nil
	})
	if !errors.Is(err, ErrFileChanged) {
		t.Fatalf("expected ErrFileChanged, got %v", err)
	}
}

func TestSeedDemoAndFinalizeBuildSearchEntries(t *testing.T) {
	store := testStore(t)
	if err := store.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.Finalize(context.Background(), "demo", time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	initialized, source, err := store.Initialized(context.Background())
	if err != nil || !initialized || source != "demo" {
		t.Fatalf("initialized=%v source=%q err=%v", initialized, source, err)
	}
	var houses, apartments int
	if err := store.pool.QueryRow(context.Background(), `SELECT count(*) FILTER (WHERE object_kind='house'), count(*) FILTER (WHERE object_kind='apartment') FROM gar_search_addresses`).Scan(&houses, &apartments); err != nil {
		t.Fatal(err)
	}
	if houses < 2 || apartments < 3 {
		t.Fatalf("houses=%d apartments=%d", houses, apartments)
	}
	var address string
	if err := store.pool.QueryRow(context.Background(), `SELECT full_address FROM gar_search_addresses WHERE object_kind='apartment' ORDER BY object_id LIMIT 1`).Scan(&address); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(address, "Москва") || !strings.Contains(address, "кв.") {
		t.Fatalf("unexpected full address %q", address)
	}
}

func TestLegacyGARVersionConstraintsAreRemoved(t *testing.T) {
	for _, constraint := range []string{
		"gar_address_objects_pkey",
		"gar_address_objects_object_guid_key",
		"gar_adm_hierarchy_pkey",
		"gar_houses_pkey",
		"gar_houses_object_guid_key",
	} {
		if !strings.Contains(schemaSQL, "DROP CONSTRAINT IF EXISTS "+constraint) {
			t.Errorf("schema does not remove legacy constraint %s", constraint)
		}
	}
	if strings.Contains(indexesSQL, "CREATE UNIQUE INDEX IF NOT EXISTS gar_address_objects_object_id") {
		t.Error("OBJECTID must not be unique because GAR contains historical versions")
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("GAR_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("GAR_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("gar_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	admin.Close()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	store := New(pool)
	if err := store.EnsureSchema(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, err := pgxpool.New(ctx, url)
		if err == nil {
			_, _ = cleanup.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
			cleanup.Close()
		}
	})
	return store
}

func emptyRow(family model.Family) []any {
	row := make([]any, len(family.Columns()))
	row[len(row)-1] = []byte(`{}`)
	return row
}

func familyByKey(t *testing.T, key string) model.Family {
	t.Helper()
	for _, family := range model.Families() {
		if family.Key == key {
			return family
		}
	}
	t.Fatalf("unknown family %s", key)
	return model.Family{}
}

func columnIndex(family model.Family, column string) int {
	for index, candidate := range family.Columns() {
		if candidate == column {
			return index
		}
	}
	return -1
}

func assertCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count: want %d, got %d", table, want, got)
	}
}
