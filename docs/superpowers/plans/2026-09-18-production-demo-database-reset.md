# Production Demo Database Reset Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the full GAR/FIAS production database with a reproducible Arbat demo fixture that is identical locally and in production and is verified against production before deletion.

**Architecture:** `gar-init` gains an explicit `demo` mode and loads a set of canonical CSV fixtures through PostgreSQL `COPY`; `full` mode retains the existing XML importer. Read-only export and parity scripts create the fixture and manifests from one repeatable-read production snapshot, while a separate guarded reset script performs the destructive schema reset only after parity and backup checks pass.

**Tech Stack:** Go 1.25, pgx v5, PostgreSQL 17, POSIX shell, Docker Compose

**Spec:** `docs/superpowers/specs/2026-09-18-production-demo-database-reset-design.md`

## Global Constraints

- Local and production Compose must explicitly select `GAR_MODE=demo`.
- Demo mode must never inspect or import the mounted GAR XML directory.
- The fixture contains exactly ten real Arbat houses and representative related rows from every available GAR family.
- Production rows and locally loaded fixture rows must compare byte-for-byte through canonical CSV exports and SHA-256 manifests.
- No production deletion may occur before fixture validation and backup completion.
- The destructive reset remains an explicit operator command and never runs during normal startup or deploy.
- Existing production requests, photos, caches, and the full GAR snapshot are intentionally deleted by the approved reset.

---

### Task 1: Explicit GAR operating mode

**Files:**
- Modify: `gar-init/internal/config/config.go`
- Modify: `gar-init/internal/config/config_test.go`
- Modify: `gar-init/internal/importer/run.go`
- Modify: `gar-init/internal/importer/run_test.go`

**Interfaces:**
- Produces: `config.Mode` with constants `ModeDemo` and `ModeFull`.
- Produces: `Config.Mode`, parsed from required `GAR_MODE` values `demo` or `full`.
- Consumes: existing `Store.SeedDemo` and XML import interfaces.

- [ ] **Step 1: Write failing configuration tests**

Add table-driven cases asserting that `GAR_MODE=demo` and `GAR_MODE=full` parse to their constants, an empty value fails with `GAR_MODE is required`, and any other value fails with `GAR_MODE must be demo or full`.

- [ ] **Step 2: Run the configuration tests and verify the expected compile or assertion failure**

Run: `cd gar-init && go test ./internal/config -run GARMode -count=1`

Expected: FAIL because `Config.Mode`, `ModeDemo`, and `ModeFull` do not exist.

- [ ] **Step 3: Implement strict mode parsing**

Define:

```go
type Mode string

const (
    ModeDemo Mode = "demo"
    ModeFull Mode = "full"
)
```

Add `Mode Mode` to `Config`, require `GAR_MODE`, and reject values outside the two constants. Remove `AllowDemoData` and its boolean parsing.

- [ ] **Step 4: Write failing importer tests for mode isolation**

Add tests proving demo mode seeds and finalizes without calling `Discover`, while full mode with an empty directory returns `ErrNoSupportedFiles` and never seeds demo data.

- [ ] **Step 5: Run importer tests and verify the demo-mode test fails because discovery still runs**

Run: `cd gar-init && go test ./internal/importer -run Mode -count=1`

Expected: FAIL with demo mode attempting file discovery or the old fallback behavior.

- [ ] **Step 6: Route importer behavior by mode**

In `Run`, after the initialized check, branch on `cfg.Mode`. For `ModeDemo`, call `SeedDemo`, finalize with source type `demo`, log completion, and return before `Discover`. For `ModeFull`, execute only the existing XML path and propagate missing/incomplete snapshot errors.

- [ ] **Step 7: Run focused and package tests**

Run: `cd gar-init && go test ./internal/config ./internal/importer -count=1`

Expected: PASS.

### Task 2: Canonical CSV fixture loader

**Files:**
- Create: `gar-init/internal/db/fixture.go`
- Create: `gar-init/internal/db/fixtures/*.csv`
- Modify: `gar-init/internal/db/demo.go`
- Modify: `gar-init/internal/db/store_test.go`

**Interfaces:**
- Produces: `fixtureTables []fixtureTable`, each entry containing an exact table name, ordered column list, and embedded CSV bytes.
- Produces: `(*Store).SeedDemo(context.Context) error` backed by server-side CSV parsing through `COPY ... FROM STDIN WITH (FORMAT csv, HEADER true)`.
- Consumes: the existing GAR schema and transaction boundary.

- [ ] **Step 1: Write a failing database test for all fixture families**

Extend `TestSeedDemoAndFinalizeBuildSearchEntries` to assert exact nonzero counts for address objects, houses, address/object parameters, both hierarchies when present, apartments, rooms, car places, reestr objects, change history, normative documents, and the rebuilt search table. Assert ten houses after the real fixture replaces the bootstrap fixture.

- [ ] **Step 2: Run the integration test and verify it fails on missing fixture families**

Run: `cd gar-init && GAR_TEST_DATABASE_URL='postgres://tvoydom:tvoydom_demo@localhost:5433/tvoydom?sslmode=disable' go test ./internal/db -run TestSeedDemoAndFinalizeBuildSearchEntries -count=1`

Expected: FAIL because the existing hard-coded demo inserts do not populate the required families.

- [ ] **Step 3: Add embedded CSV table definitions**

Create `fixture.go` with `//go:embed fixtures/*.csv`, a fixed ordered list covering every GAR source table except `gar_import_state`, `gar_import_metadata`, and derived `gar_search_addresses`, and a helper that builds a quoted `COPY table (columns...) FROM STDIN WITH (FORMAT csv, HEADER true)` statement from trusted constants.

- [ ] **Step 4: Replace hard-coded inserts with transactional CSV loading**

Keep the existing empty-database guard. For each table, open its embedded CSV and call `tx.Conn().PgConn().CopyFrom`; reject a missing header, malformed CSV, or a table whose loaded row count differs from its data-line count. Commit only after every table loads.

- [ ] **Step 5: Add a small complete bootstrap fixture**

Represent the current synthetic Moscow/Arbat data in CSV and add minimal linked rows for currently empty families. Mark it as bootstrap-only in its manifest; Task 6 replaces it with real production rows before release.

- [ ] **Step 6: Run database tests**

Run: `cd gar-init && GAR_TEST_DATABASE_URL='postgres://tvoydom:tvoydom_demo@localhost:5433/tvoydom?sslmode=disable' go test ./internal/db -count=1`

Expected: PASS.

### Task 3: Read-only production extractor and parity manifests

**Files:**
- Create: `scripts/export-arbat-fixture.sql`
- Create: `scripts/export-arbat-fixture.sh`
- Create: `scripts/export-demo-database.sql`
- Create: `scripts/verify-arbat-fixture.sh`
- Create: `gar-init/internal/db/fixtures/manifest.sha256`
- Create: `gar-init/internal/db/fixtures/source.json`
- Create: `scripts/tests/fixture-scripts_test.sh`

**Interfaces:**
- Produces: `export-arbat-fixture.sh DATABASE_URL OUTPUT_DIR HOUSE_IDS`, where `HOUSE_IDS` is ten comma-separated positive GAR object IDs.
- Produces: canonical CSV files with schema-order columns, deterministic ordering, CSV headers, and PostgreSQL `NULL` represented as an unquoted empty field only where the CSV parser preserves null distinction through `FORCE_NOT_NULL` exclusions documented per table.
- Produces: `source.json` with database name, source date, extraction timestamp, street ID, house IDs, per-table counts, and hashes; credentials are excluded.
- Produces: `verify-arbat-fixture.sh DATABASE_URL FIXTURE_DIR`, returning nonzero for any missing, extra, or changed row.

- [ ] **Step 1: Write failing shell contract tests**

The test script checks that export SQL starts `BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY`, selects one Moscow Arbat street, validates exactly ten supplied houses, creates the complete retained-object closure, exports every fixture table with explicit columns and ordering, and commits only after all exports. It also checks that manifests omit strings matching `postgres://`, `password`, and `POSTGRES_PASSWORD`.

- [ ] **Step 2: Run the shell contract test and verify missing scripts fail**

Run: `sh scripts/tests/fixture-scripts_test.sh`

Expected: FAIL because the export and verification scripts do not exist.

- [ ] **Step 3: Implement the single-snapshot export SQL**

Use psql variables for output directory and the ten house IDs. Inside one repeatable-read, read-only transaction, build temporary retained-ID tables, select bounded descendants with `row_number()` per parent, close required ancestors recursively, and use `\copy (SELECT explicit_columns ... ORDER BY logical_keys) TO :'file' WITH (FORMAT csv, HEADER true)` for all source tables.

- [ ] **Step 4: Implement manifest generation**

The wrapper validates arguments, creates a private temporary output directory, invokes psql with `ON_ERROR_STOP=1`, calculates portable SHA-256 values using `sha256sum` or `shasum -a 256`, writes counts and non-secret source metadata, and atomically renames the directory only after success.

- [ ] **Step 5: Implement local parity export and comparison**

`export-demo-database.sql` exports every GAR source table from the loaded demo database with exactly the same explicit columns and ordering. `verify-arbat-fixture.sh` compares filenames, line counts, and hashes and prints the first differing table without printing credentials.

- [ ] **Step 6: Run script tests**

Run: `sh scripts/tests/fixture-scripts_test.sh`

Expected: PASS.

### Task 4: Guarded production reset command

**Files:**
- Create: `scripts/reset-production-demo.sh`
- Modify: `scripts/tests/fixture-scripts_test.sh`

**Interfaces:**
- Produces: `reset-production-demo.sh EXPECTED_DATABASE BACKUP_DIR FIXTURE_DIR` for execution on the production VM from the repository root.
- Consumes: production Compose environment, the committed fixture manifest, and the parity scripts.

- [ ] **Step 1: Add failing reset safety tests**

Check that the script requires all three arguments, verifies the live database name exactly, stops backend and `gar-init`, reruns the production export, compares manifests, creates and verifies a nonempty custom-format `pg_dump`, requires the exact typed phrase `RESET <database>`, and uses `ON_ERROR_STOP=1` for schema recreation.

- [ ] **Step 2: Run the shell tests and verify failure on the missing reset script**

Run: `sh scripts/tests/fixture-scripts_test.sh`

Expected: FAIL because `reset-production-demo.sh` does not exist.

- [ ] **Step 3: Implement guarded reset**

Resolve Compose variables without echoing the password. Abort on database-name mismatch, parity mismatch, insufficient backup creation, failed dump verification, or confirmation mismatch. Execute the destructive SQL only after all gates:

```sql
DROP SCHEMA public CASCADE;
CREATE SCHEMA public AUTHORIZATION CURRENT_USER;
GRANT ALL ON SCHEMA public TO CURRENT_USER;
GRANT USAGE ON SCHEMA public TO PUBLIC;
```

Then run `gar-init`, start backend/frontend, and leave traffic unavailable if any initialization or smoke check fails.

- [ ] **Step 4: Run shell tests**

Run: `sh scripts/tests/fixture-scripts_test.sh`

Expected: PASS.

### Task 5: Compose, deployment, and documentation

**Files:**
- Modify: `docker-compose.yml`
- Modify: `docker-compose.prod.yml`
- Modify: `.env.example`
- Modify: `README.md`
- Modify: `gar-init/README.md`
- Modify: `deploy.sh`

**Interfaces:**
- Consumes: `GAR_MODE=demo`.
- Produces: local and production service ordering `postgres -> gar-init -> backend -> frontend`.

- [ ] **Step 1: Add configuration assertions to the shell contract test**

Assert both Compose files set `GAR_MODE: demo`, production `gar-init` is no longer hidden behind the init profile, backend depends on successful `gar-init`, and neither Compose file exposes the old `GAR_ALLOW_DEMO_DATA` fallback.

- [ ] **Step 2: Run the shell test and verify the configuration assertions fail**

Run: `sh scripts/tests/fixture-scripts_test.sh`

Expected: FAIL on the old Compose configuration.

- [ ] **Step 3: Update runtime configuration**

Set explicit demo mode in both Compose files. Make production `gar-init` an idempotent prerequisite for backend. Ensure normal deploy pulls and starts the `gar-init` image before backend without mounting or scanning full XML in demo mode.

- [ ] **Step 4: Update operator documentation**

Document demo/full modes, fixture provenance, parity commands, backup artifacts, exact reset ordering, rollback, and the fact that normal local/production startup contains only the Arbat fixture.

- [ ] **Step 5: Validate rendered Compose files**

Run: `docker compose config >/tmp/max-compose.yml && POSTGRES_PASSWORD=test docker compose -f docker-compose.prod.yml config >/tmp/max-compose-prod.yml && sh scripts/tests/fixture-scripts_test.sh`

Expected: PASS.

### Task 6: Extract and validate the real Arbat fixture

**Files:**
- Replace: `gar-init/internal/db/fixtures/*.csv`
- Replace: `gar-init/internal/db/fixtures/manifest.sha256`
- Replace: `gar-init/internal/db/fixtures/source.json`

**Interfaces:**
- Consumes: read-only production database access and the existing GIS Housing client.
- Produces: the release fixture containing ten real, verified Arbat houses and related rows.

- [ ] **Step 1: Inspect production candidates without writes**

Query the unique Moscow Arbat street and candidate houses with counts of apartments, rooms, car places, params, reestr rows, and history. Select ten house IDs that maximize family coverage.

- [ ] **Step 2: Verify real GIS Housing resolution**

Run the existing GIS client against candidate GUIDs and retain at least two houses that return successful profiles. Record only house IDs and verification status; do not commit response bodies.

- [ ] **Step 3: Export the production fixture**

Run `scripts/export-arbat-fixture.sh` using the ten selected house IDs and copy the completed output off the production VM before any deletion.

- [ ] **Step 4: Replace the bootstrap fixture and run local parity validation**

Load the fixture into a clean local PostgreSQL schema through `gar-init`, rebuild search, export the loaded rows, and run `verify-arbat-fixture.sh`. Verify exact counts and hashes for every table.

- [ ] **Step 5: Run application smoke checks locally**

Verify address search, parent filtering, address lookup, two successful `/api/houses/resolve` calls, and request creation with a sampled apartment.

### Task 7: Full verification and production reset

**Files:**
- Modify only if verification reveals a tested defect in files above.

**Interfaces:**
- Consumes: validated release fixture, production backup location, and approved destructive reset.
- Produces: a small production demo database and a final commit on `main`.

- [ ] **Step 1: Run complete local verification**

Run:

```bash
cd gar-init && go test ./... && go vet ./...
cd ../backend && go test ./... && go vet ./...
cd ../frontend && pnpm build
cd .. && docker compose config >/tmp/max-compose.yml
POSTGRES_PASSWORD=test docker compose -f docker-compose.prod.yml config >/tmp/max-compose-prod.yml
sh scripts/tests/fixture-scripts_test.sh
git diff --check
```

Expected: every command exits zero.

- [ ] **Step 2: Commit implementation and fixture on `main`**

Stage only files belonging to this plan and commit with message `feat: replace full GAR with verified Arbat demo data`.

- [ ] **Step 3: Recheck production parity immediately before reset**

Regenerate the source export and compare it with the committed fixture. Abort if any selected row changed.

- [ ] **Step 4: Execute the guarded production reset**

Run the reset command with the exact production database name, an external backup directory, and the committed fixture directory. Type the required confirmation only after the script reports matching parity and a verified backup.

- [ ] **Step 5: Verify production behavior and size**

Run production health, address search, house resolution, request creation, per-table counts, and database-size checks. Confirm that only the Arbat demo scope remains and retain the dump, canonical exports, and matching manifests outside the PostgreSQL volume.
