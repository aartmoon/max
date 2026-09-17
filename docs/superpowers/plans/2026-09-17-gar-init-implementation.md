# GAR Init and Address Autocomplete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy GAR ZIP importer with a standalone restart-safe XML initialization job and wire its house/apartment data into the backend API and request form autocomplete.

**Architecture:** A separate Go module streams each mounted XML file into a dedicated PostgreSQL table with bounded `pgx.CopyFrom` batches and per-file transactions. After all files commit, it builds an indexed search table; the backend validates selected GAR IDs and the frontend requires a GAR house selection with an optional apartment.

**Tech Stack:** Go 1.25+, `encoding/xml`, pgx v5, PostgreSQL 17, Docker Compose, React 19, TypeScript, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-17-gar-init-design.md`

## Global Constraints

- Parse 1-3 GB XML files only with `encoding/xml.Decoder`; never unmarshal a whole document.
- Keep memory bounded by `GAR_BATCH_SIZE`, default `10000` rows.
- Use pgx v5 and `CopyFrom`; never issue one `INSERT` per imported row.
- Use one transaction per XML file and write its state row in the same transaction.
- Never use `DROP TABLE` or `TRUNCATE` in the importer.
- Hold a PostgreSQL session advisory lock for the entire initialization.
- Build secondary indexes only after all raw files import successfully.
- A completed database must log `GAR database already initialized` and exit 0.
- New requests require a valid active GAR house; apartment selection remains optional.

---

### Task 1: Standalone module, configuration, and file manifest

**Files:**
- Create: `gar-init/go.mod`
- Create: `gar-init/internal/config/config.go`
- Create: `gar-init/internal/config/config_test.go`
- Create: `gar-init/internal/model/family.go`
- Create: `gar-init/internal/importer/discovery.go`
- Create: `gar-init/internal/importer/discovery_test.go`

**Interfaces:**
- Produces: `config.Load(getenv func(string) string) (config.Config, error)`.
- Produces: `model.Families() []model.Family` and `model.MatchFile(name string) (model.Family, bool)`.
- Produces: `importer.Discover(dir string) ([]importer.SourceFile, error)`.

- [ ] **Step 1: Write failing configuration tests**

```go
func TestLoadRequiresPostgresSettings(t *testing.T) {
    _, err := Load(func(string) string { return "" })
    if err == nil || !strings.Contains(err.Error(), "POSTGRES_HOST") {
        t.Fatalf("expected POSTGRES_HOST error, got %v", err)
    }
}

func TestLoadAppliesGARDefaults(t *testing.T) {
    values := map[string]string{
        "POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432",
        "POSTGRES_DB": "app", "POSTGRES_USER": "app", "POSTGRES_PASSWORD": "secret",
    }
    got, err := Load(func(k string) string { return values[k] })
    if err != nil { t.Fatal(err) }
    if got.DataDir != "/data/gar" || got.BatchSize != 10000 || got.LogEvery != 100000 {
        t.Fatalf("unexpected defaults: %+v", got)
    }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run: `cd gar-init && go test ./internal/config ./internal/importer`

Expected: failure because packages and functions do not exist.

- [ ] **Step 3: Implement configuration and the exact 18-family manifest**

`Config` contains host, port, database, user, password, data directory, batch size, and log interval. `Family` contains key, filename prefix, table, accepted child element names, columns, and deterministic order. Matching tests must prove longer prefixes win, including `AS_ADDR_OBJ_PARAMS_` before `AS_ADDR_OBJ_` and `AS_HOUSES_PARAMS_` before `AS_HOUSES_`.

- [ ] **Step 4: Implement sorted discovery and missing-family validation**

Use `os.ReadDir`, accept `.XML` case-insensitively, reject duplicate files for one family, report every missing family in one error, and sort by manifest order rather than filename.

- [ ] **Step 5: Run tests and verify GREEN**

Run: `cd gar-init && go test ./internal/config ./internal/importer`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add gar-init/go.mod gar-init/internal/config gar-init/internal/model gar-init/internal/importer
git commit -m "feat: define GAR initializer manifest"
```

### Task 2: Streaming XML parser and bounded batches

**Files:**
- Create: `gar-init/internal/model/value.go`
- Create: `gar-init/internal/importer/parser.go`
- Create: `gar-init/internal/importer/parser_test.go`
- Create: `gar-init/internal/importer/testdata/houses.xml`
- Create: `gar-init/internal/importer/testdata/apartments.xml`

**Interfaces:**
- Produces: `type BatchWriter func(context.Context, model.Family, [][]any) error`.
- Produces: `Parse(ctx context.Context, r io.Reader, family model.Family, batchSize, logEvery int, write BatchWriter, progress func(int64)) (int64, error)`.
- Consumes: `model.Family` column definitions from Task 1.

- [ ] **Step 1: Write failing parser tests**

```go
func TestParseStreamsHouseAttributesInBoundedBatches(t *testing.T) {
    xmlData := `<HOUSES><HOUSE OBJECTID="10" OBJECTGUID="11111111-2222-3333-4444-555555555555" HOUSENUM="7" BUILDNUM="2" ISACTUAL="1" ISACTIVE="1"/><HOUSE OBJECTID="11" HOUSENUM="8" ISACTUAL="0" ISACTIVE="1"/></HOUSES>`
    var sizes []int
    count, err := Parse(context.Background(), strings.NewReader(xmlData), mustFamily(t, "houses"), 1, 100000,
        func(_ context.Context, _ model.Family, rows [][]any) error {
            sizes = append(sizes, len(rows))
            return nil
        }, func(int64) {})
    if err != nil { t.Fatal(err) }
    if count != 2 || !reflect.DeepEqual(sizes, []int{1, 1}) {
        t.Fatalf("count=%d sizes=%v", count, sizes)
    }
}

func TestParseRejectsMalformedTypedAttribute(t *testing.T) {
    input := `<HOUSES><HOUSE OBJECTID="not-a-number"/></HOUSES>`
    _, err := Parse(context.Background(), strings.NewReader(input), mustFamily(t, "houses"), 10, 100, noOpWriter, func(int64) {})
    if err == nil || !strings.Contains(err.Error(), "OBJECTID") { t.Fatalf("got %v", err) }
}
```

Also test `OBJECT`, `ITEM`, `APARTMENT`, `PARAM`, cancellation, malformed XML, boolean variants `0/1/true/false`, nullable UUID/date fields, raw JSON attributes, and a final partial batch.

- [ ] **Step 2: Run tests and verify RED**

Run: `cd gar-init && go test ./internal/importer -run 'TestParse'`

Expected: failure because `Parse` does not exist.

- [ ] **Step 3: Implement token streaming and typed conversion**

Call `decoder.Token()` in a loop, process only accepted `xml.StartElement` names, convert declared columns, and encode all attributes to JSON. Include `decoder.InputOffset()` in conversion and syntax errors. Reuse `rows[:0]` after each write so capacity never grows beyond the configured batch.

- [ ] **Step 4: Run parser tests and verify GREEN**

Run: `cd gar-init && go test ./internal/importer -run 'TestParse'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gar-init/internal/model gar-init/internal/importer
git commit -m "feat: stream GAR XML in bounded batches"
```

### Task 3: PostgreSQL schema, state, and transactional COPY

**Files:**
- Create: `gar-init/internal/db/schema.sql`
- Create: `gar-init/internal/db/indexes.sql`
- Create: `gar-init/internal/db/search.sql`
- Create: `gar-init/internal/db/store.go`
- Create: `gar-init/internal/db/store_test.go`

**Interfaces:**
- Produces: `db.New(pool *pgxpool.Pool) *db.Store`.
- Produces: `EnsureSchema`, `AcquireLock`, `Initialized`, `FileState`, `ImportFile`, and `Finalize` methods.
- `ImportFile(ctx, SourceFile, func(context.Context, BatchWriter) (int64, error)) error` owns the per-file transaction.

- [ ] **Step 1: Write failing PostgreSQL integration tests**

Tests skip only when `GAR_TEST_DATABASE_URL` is empty. When configured, each test uses a unique schema through `search_path` and verifies:

```go
func TestImportFileRollsBackRowsAndStateTogether(t *testing.T) {
    store := testStore(t)
    source := importer.SourceFile{Name: "AS_HOUSES_test.XML", Size: 123, Family: mustFamily(t, "houses")}
    errBoom := errors.New("copy failed")
    err := store.ImportFile(context.Background(), source, func(ctx context.Context, write importer.BatchWriter) (int64, error) {
        if err := write(ctx, source.Family, [][]any{validHouseRow()}); err != nil { return 0, err }
        return 1, errBoom
    })
    if !errors.Is(err, errBoom) { t.Fatalf("got %v", err) }
    assertCount(t, store, "gar_houses", 0)
    assertCount(t, store, "gar_import_state", 0)
}
```

Add tests for successful state insertion, completed-file skip, same-name size mismatch, initialized shortcut, lock contention, finalization idempotence, and expected indexes.

- [ ] **Step 2: Run integration tests and verify RED**

Run: `cd gar-init && GAR_TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/db -count=1`

Expected: failure because store and schema do not exist.

- [ ] **Step 3: Write complete idempotent DDL**

Create the two state tables, all 18 raw family tables, and `gar_search_addresses`. Include typed columns from the spec and `raw_attributes jsonb NOT NULL`. Do not create secondary indexes in `schema.sql`; place them in `indexes.sql` with `CREATE INDEX IF NOT EXISTS`.

- [ ] **Step 4: Implement lock, state, transactions, and CopyFrom**

Acquire a dedicated pooled connection and call `SELECT pg_advisory_lock($1)` with one constant key. `ImportFile` begins a transaction, adapts batches to `pgx.CopyFromRows`, calls `tx.CopyFrom`, inserts state, and commits. A deferred rollback handles every failure.

- [ ] **Step 5: Implement transactional finalization**

Execute `indexes.sql`, rebuild `gar_search_addresses` with recursive hierarchy SQL from `search.sql`, then upsert the singleton completion marker. Avoid destructive table operations by using `INSERT ... ON CONFLICT DO UPDATE` and an empty search table on first initialization.

- [ ] **Step 6: Run integration tests and verify GREEN**

Run: `cd gar-init && GAR_TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/db -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add gar-init/internal/db
git commit -m "feat: add restart-safe GAR PostgreSQL store"
```

### Task 4: Import orchestration, CLI, and container

**Files:**
- Create: `gar-init/internal/importer/run.go`
- Create: `gar-init/internal/importer/run_test.go`
- Create: `gar-init/cmd/gar-init/main.go`
- Create: `gar-init/Dockerfile`
- Create: `gar-init/.dockerignore`

**Interfaces:**
- Produces: `importer.Run(ctx context.Context, cfg config.Config, store importer.Store, logger *log.Logger) error`.
- Consumes: discovery/parser from Tasks 1-2 and database store from Task 3.

- [ ] **Step 1: Write failing orchestration tests**

Use a fake store to prove initialized runs do not discover/open files, completed files are skipped, progress logs occur at exact multiples, failure stops later families, finalization happens once, and expected log lines contain filename, family, rows, and duration.

- [ ] **Step 2: Run tests and verify RED**

Run: `cd gar-init && go test ./internal/importer -run 'TestRun'`

Expected: failure because orchestration does not exist.

- [ ] **Step 3: Implement orchestration and signal-aware main**

`main` loads environment variables, creates a signal context, connects with pgxpool, pings PostgreSQL, ensures schema, acquires the advisory lock, calls `Run`, and exits non-zero on error. Never wrap the multi-hour import in a short global timeout.

- [ ] **Step 4: Add a multi-stage Dockerfile**

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gar-init ./cmd/gar-init

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
USER app
COPY --from=build /out/gar-init /usr/local/bin/gar-init
ENTRYPOINT ["/usr/local/bin/gar-init"]
```

- [ ] **Step 5: Run unit tests and build**

Run: `cd gar-init && go test ./... && go build ./cmd/gar-init`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add gar-init
git commit -m "feat: add GAR initialization job"
```

### Task 5: Compose lifecycle integration

**Files:**
- Modify: `docker-compose.yml`
- Modify: `docker-compose.prod.yml`
- Create: `.env.example`

**Interfaces:**
- Produces: `gar-init` Compose service and backend completion dependency.

- [ ] **Step 1: Record the failing Compose expectation**

Run: `docker compose config | rg 'gar-init|service_completed_successfully|/data/gar'`

Expected: no `gar-init` service before the change.

- [ ] **Step 2: Add local and production services**

Use `build: ./gar-init` locally, the same PostgreSQL environment variables as the database, `${GAR_XML_PATH:-/home/user1/gar/xml}:/data/gar:ro`, `restart: "no"`, and the private backend network in production. Make backend depend on both healthy PostgreSQL and successful `gar-init` completion.

- [ ] **Step 3: Validate rendered Compose**

Run: `docker compose config >/tmp/max-compose.yml && docker compose -f docker-compose.prod.yml config >/tmp/max-compose-prod.yml && rg 'service_completed_successfully|/data/gar' /tmp/max-compose.yml /tmp/max-compose-prod.yml`

Expected: both files render and contain the dependency and read-only mount.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml docker-compose.prod.yml .env.example
git commit -m "chore: run GAR initializer before backend"
```

### Task 6: Backend schema and GAR repository

**Files:**
- Remove: `backend/cmd/gar-import/main.go`
- Remove: `backend/integration/gar/importer.go`
- Remove: `backend/integration/gar/importer_test.go`
- Replace: `backend/repository/migrations/003_gar_house_identity.sql`
- Create: `backend/repository/migrations/004_gar_address_selection.sql`
- Modify: `backend/repository/postgres.go`
- Replace: `backend/repository/address.go`
- Modify: `backend/repository/address_test.go`
- Modify: `backend/domain/models.go`

**Interfaces:**
- Produces: `SearchAddresses(ctx, domain.AddressSearch) ([]domain.AddressSuggestion, error)`.
- Produces: `GetAddress(ctx, objectID int64) (domain.AddressInfo, error)`.
- Produces: `ResolveHouse(ctx, domain.AddressInfo) (domain.House, error)`.
- Produces domain types with numeric-string JSON IDs to avoid JavaScript integer precision loss.

- [ ] **Step 1: Write failing repository/domain tests**

Test normalization, query validation, prefix-before-trigram SQL behavior, kind filtering, parent filtering, active-only retrieval, and house upsert by `gar_object_id`. The expected suggestion shape is:

```go
domain.AddressSuggestion{
    ObjectID: "123", ObjectGUID: "11111111-2222-3333-4444-555555555555",
    ParentObjectID: "100", ObjectKind: "house",
    DisplayName: "д. 7", FullAddress: "г. Москва, ул. Тверская, д. 7",
}
```

- [ ] **Step 2: Run backend tests and verify RED**

Run: `cd backend && go test ./domain ./repository`

Expected: compile/test failure for the new contracts.

- [ ] **Step 3: Add migration 004 and register it**

Add `gar_object_id bigint` and `object_guid uuid` to `houses`; add immutable request columns `address_snapshot text`, `house_object_id bigint`, `house_object_guid uuid`, `apartment_object_id bigint`, and `apartment_object_guid uuid`. Backfill existing demo requests from their joined house address, then make `address_snapshot` non-null. Register migration version 4 in `Migrate`.

- [ ] **Step 4: Replace legacy repository code**

Query `gar_search_addresses` directly, validate numeric object IDs, use parameterized filters, return empty arrays rather than null, and upsert an application house from a validated GAR house. Remove all importer-specific repository methods and the recursive refresh SQL now owned by `gar-init`.

- [ ] **Step 5: Run repository tests and verify GREEN**

Run: `cd backend && go test ./domain ./repository`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add -A backend
git commit -m "feat: query initialized GAR address index"
```

### Task 7: Backend address APIs and request validation

**Files:**
- Replace: `backend/service/house.go`
- Modify: `backend/service/house_test.go`
- Modify: `backend/service/request.go`
- Modify: `backend/service/request_test.go`
- Modify: `backend/controller/http.go`
- Create: `backend/controller/address_test.go`
- Modify: `backend/repository/postgres.go`

**Interfaces:**
- Search endpoint: `GET /api/addresses/search?q=&kind=house&parentObjectId=&limit=10`.
- Detail endpoint: `GET /api/addresses/{objectId}`.
- Resolve endpoint: `POST /api/houses/resolve` with `{"objectId":"123"}`.
- Request fields: `houseObjectId` required, `apartmentObjectId` optional.

- [ ] **Step 1: Write failing service tests**

```go
func TestCreateRejectsApartmentFromAnotherHouse(t *testing.T) {
    addresses := fakeAddresses{
        "10": {ObjectID: "10", ObjectKind: "house", IsActive: true},
        "20": {ObjectID: "20", ParentObjectID: "11", ObjectKind: "apartment", IsActive: true},
    }
    svc := validRequestService(addresses)
    _, err := svc.Create(context.Background(), CreateInput{Description: "Течёт труба", Kind: "PROBLEM", HouseObjectID: "10", ApartmentObjectID: "20"})
    if err == nil || !strings.Contains(err.Error(), "квартира") { t.Fatalf("got %v", err) }
}
```

Also prove arbitrary text cannot create a request, inactive/wrong-kind houses fail, a valid apartment produces its `full_address` snapshot, and a house-only request uses the house address.

- [ ] **Step 2: Write failing HTTP contract tests**

Test invalid `limit`, invalid `kind`, missing `q` without parent, malformed object ID, JSON error shape, resolve body validation, and multipart parsing of the two GAR IDs.

- [ ] **Step 3: Run tests and verify RED**

Run: `cd backend && go test ./service ./controller`

Expected: FAIL for the new contracts.

- [ ] **Step 4: Implement services and handlers**

Resolve IDs server-side through the address repository, enforce kind/active/parent constraints, derive `Address`, persist the GAR identifiers and snapshot, and return them in request JSON. Update SQL inserts/selects/scans consistently.

- [ ] **Step 5: Run all backend tests and verify GREEN**

Run: `cd backend && go test ./...`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend
git commit -m "feat: validate GAR address selections in requests"
```

### Task 8: Frontend house and apartment autocomplete

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Create: `frontend/src/components/AddressAutocomplete.tsx`
- Modify: `frontend/src/pages/NewRequest.tsx`
- Modify: `frontend/src/styles.css`
- Create: `frontend/tests/address-autocomplete.spec.ts`
- Modify: relevant existing Playwright request fixtures under `frontend/tests/`

**Interfaces:**
- Produces: `addressApi.search({q, kind, parentObjectId, signal})`.
- Produces: `AddressAutocomplete` controlled selection component.
- Submits string IDs as multipart `houseObjectId` and optional `apartmentObjectId`.

- [ ] **Step 1: Write failing Playwright tests**

Intercept `/max/api/addresses/search*` and `/max/api/requests`. Verify house results appear only after debounce, keyboard ArrowDown/Enter selects a house, apartment calls contain the selected parent ID, editing a selected house clears the apartment, unselected free text blocks submission, and the multipart body contains the selected IDs.

- [ ] **Step 2: Run the focused browser test and verify RED**

Run: `cd frontend && pnpm test:e2e -- address-autocomplete.spec.ts`

Expected: FAIL because the component and API client do not exist.

- [ ] **Step 3: Add typed API methods and accessible component**

Use combobox/listbox roles, `aria-expanded`, `aria-controls`, `aria-activedescendant`, pointer selection without input blur races, a 300 ms effect timer, and `AbortController`. Show explicit loading, empty, and retryable error messages.

- [ ] **Step 4: Replace free-text request address**

Require a selected house, reveal the apartment selector afterward, derive the displayed full address from the selection, remove the test-address shortcut, and append only the selected GAR IDs to the multipart request.

- [ ] **Step 5: Run build and all browser tests**

Run: `cd frontend && pnpm build && pnpm test:e2e`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend
git commit -m "feat: select GAR house and apartment in request form"
```

### Task 9: Operational documentation and end-to-end verification

**Files:**
- Modify: `README.md`
- Create: `gar-init/README.md`

**Interfaces:**
- Documents build, first run, progress, row-count checks, failure restart, initialized restart, and schema queries.

- [ ] **Step 1: Document exact operating commands**

Include:

```bash
GAR_XML_PATH=/home/user1/gar/xml docker compose up --build gar-init
docker compose up --build -d
docker compose logs -f gar-init
docker compose run --rm gar-init
```

Include verification SQL for every raw table, state rows, final metadata, and search kinds. Explain that restarting after a failed file is safe because the file transaction rolled back; completed files are skipped.

- [ ] **Step 2: Run formatting and static checks**

Run: `gofmt -w gar-init backend && cd gar-init && go vet ./... && cd ../backend && go vet ./...`

Expected: no output and exit 0.

- [ ] **Step 3: Run all automated verification**

Run: `cd gar-init && go test ./... && cd ../backend && go test ./... && cd ../frontend && pnpm build && pnpm test:e2e`

Expected: PASS.

- [ ] **Step 4: Validate containers and Compose**

Run: `docker build -t max-gar-init-test ./gar-init && docker compose config >/tmp/max-compose.yml && docker compose -f docker-compose.prod.yml config >/tmp/max-compose-prod.yml`

Expected: image builds and both Compose files validate.

- [ ] **Step 5: Inspect final diff and confirm forbidden operations are absent**

Run: `git diff --check && ! rg -n 'DROP TABLE|TRUNCATE|io.ReadAll|xml.Unmarshal' gar-init && git status --short`

Expected: clean diff checks, no forbidden importer operations, and only intended files changed.

- [ ] **Step 6: Commit documentation**

```bash
git add README.md gar-init/README.md
git commit -m "docs: explain GAR initialization and restart"
```
