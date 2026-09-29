package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkflowMigrationsPreserveLegacyRequestAndSeedOrganizations(t *testing.T) {
	databaseURL := os.Getenv("WORKFLOW_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("WORKFLOW_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := fmt.Sprintf("workflow_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schemaName}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	admin.Close()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ", public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgxpool.New(context.Background(), databaseURL)
		if cleanupErr == nil {
			_, _ = cleanup.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schemaName}.Sanitize()+" CASCADE")
			cleanup.Close()
		}
	})
	repo := Postgres{Pool: pool}
	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO requests(user_id,house_id,description,problem_type,responsible_organization_id,status,deadline,created_at,kind,request_text) VALUES(1,1,'Старая заявка','OTHER',1,'RESOLVED',now(),now(),'APPLICATION','Текст')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE requests SET primary_organization_id=NULL,primary_organization_name_snapshot=''`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE requests r SET primary_organization_id=r.responsible_organization_id,primary_organization_name_snapshot=o.name,routing_reason='Сохранено из прежней версии заявки',routing_source='legacy' FROM organizations o WHERE o.id=r.responsible_organization_id`); err != nil {
		t.Fatal(err)
	}
	var status, snapshot string
	if err := pool.QueryRow(ctx, `SELECT status,primary_organization_name_snapshot FROM requests ORDER BY id DESC LIMIT 1`).Scan(&status, &snapshot); err != nil {
		t.Fatal(err)
	}
	if status != "RESOLVED" || snapshot != "УК «Тестовая»" {
		t.Fatalf("legacy request changed: status=%s snapshot=%q", status, snapshot)
	}
	var suppliers, inspections int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE type_code='RESOURCE_SUPPLIER'),count(*) FILTER(WHERE type_code='HOUSING_INSPECTION') FROM organizations`).Scan(&suppliers, &inspections); err != nil {
		t.Fatal(err)
	}
	if suppliers < 1 || inspections < 1 {
		t.Fatalf("snapshot missing: suppliers=%d inspections=%d", suppliers, inspections)
	}
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migration is not idempotent: %v", err)
	}
}
