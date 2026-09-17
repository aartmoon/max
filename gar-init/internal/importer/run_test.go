package importer

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tvoydom/gar-init/internal/config"
	"tvoydom/gar-init/internal/model"
)

func TestRunStopsImmediatelyWhenInitialized(t *testing.T) {
	store := &fakeStore{initialized: true, sourceType: "xml"}
	var output bytes.Buffer
	err := Run(context.Background(), config.Config{DataDir: filepath.Join(t.TempDir(), "missing")}, store, log.New(&output, "", 0), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if store.seeded || store.finalized || len(store.imported) != 0 {
		t.Fatalf("unexpected store calls: %+v", store)
	}
	if !strings.Contains(output.String(), "GAR database already initialized") {
		t.Fatalf("unexpected log: %s", output.String())
	}
}

func TestRunSeedsDemoOnlyForEmptyInputWhenEnabled(t *testing.T) {
	store := &fakeStore{}
	var output bytes.Buffer
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{DataDir: filepath.Join(t.TempDir(), "missing"), AllowDemoData: true}
	if err := Run(context.Background(), cfg, store, log.New(&output, "", 0), func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	if !store.seeded || !store.finalized || store.finalSource != "demo" || !store.finalDate.Equal(now) {
		t.Fatalf("unexpected demo flow: %+v", store)
	}
	if !strings.Contains(output.String(), "using demo address data") {
		t.Fatalf("unexpected log: %s", output.String())
	}
}

func TestRunRejectsEmptyInputWhenDemoDisabled(t *testing.T) {
	store := &fakeStore{}
	err := Run(context.Background(), config.Config{DataDir: t.TempDir()}, store, log.New(&bytes.Buffer{}, "", 0), time.Now)
	if !errors.Is(err, ErrNoSupportedFiles) || store.seeded {
		t.Fatalf("err=%v seeded=%v", err, store.seeded)
	}
}

func TestRunNeverSeedsIncompleteSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AS_HOUSES_20260910_test.XML"), []byte(`<HOUSES/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{}
	err := Run(context.Background(), config.Config{DataDir: dir, AllowDemoData: true}, store, log.New(&bytes.Buffer{}, "", 0), time.Now)
	if !errors.Is(err, ErrIncompleteSnapshot) || store.seeded {
		t.Fatalf("err=%v seeded=%v", err, store.seeded)
	}
}

func TestRunImportsManifestLogsProgressAndFinalizesXML(t *testing.T) {
	dir := t.TempDir()
	for _, name := range allFixtureNames() {
		body := `<ROOT/>`
		if strings.HasPrefix(name, "AS_HOUSES_") && !strings.HasPrefix(name, "AS_HOUSES_PARAMS_") {
			body = `<HOUSES><HOUSE OBJECTID="1"/><HOUSE OBJECTID="2"/></HOUSES>`
		}
		name = strings.Replace(name, "_test.XML", "_20260910_test.XML", 1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := &fakeStore{}
	var output bytes.Buffer
	cfg := config.Config{DataDir: dir, BatchSize: 1, LogEvery: 1}
	if err := Run(context.Background(), cfg, store, log.New(&output, "", 0), time.Now); err != nil {
		t.Fatal(err)
	}
	if len(store.imported) != 18 || store.finalSource != "xml" || store.finalDate.Format(time.DateOnly) != "2026-09-10" {
		t.Fatalf("unexpected import flow: %+v", store)
	}
	logs := output.String()
	for _, want := range []string{"[GAR] importing AS_HOUSES_", "[GAR] AS_HOUSES: 1 rows", "[GAR] finished AS_HOUSES: 2 rows"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("missing %q in logs:\n%s", want, logs)
		}
	}
}

type fakeStore struct {
	initialized bool
	sourceType  string
	seeded      bool
	finalized   bool
	finalSource string
	finalDate   time.Time
	imported    []string
}

func (s *fakeStore) Initialized(context.Context) (bool, string, error) {
	return s.initialized, s.sourceType, nil
}

func (s *fakeStore) ImportFile(ctx context.Context, source SourceFile, parse ParseFile) (int64, bool, error) {
	s.imported = append(s.imported, source.Family.Key)
	count, err := parse(ctx, func(context.Context, model.Family, [][]any) error { return nil })
	return count, false, err
}

func (s *fakeStore) SeedDemo(context.Context) error {
	s.seeded = true
	return nil
}

func (s *fakeStore) Finalize(_ context.Context, source string, date time.Time) error {
	s.finalized = true
	s.finalSource = source
	s.finalDate = date
	return nil
}
