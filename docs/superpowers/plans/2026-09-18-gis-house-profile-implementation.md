# GIS House Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fixed demo house passport with a browser-selected GAR house whose published profile is fetched from public GIS ЖКХ JSON endpoints and cached in PostgreSQL.

**Architecture:** The existing GAR autocomplete supplies a house object ID and GUID. `HouseService` resolves that identity, consults a profile repository, refreshes stale or missing profiles through an isolated GIS ЖКХ HTTP adapter, and returns one normalized response; the frontend persists only the GAR object ID in local storage.

**Tech Stack:** Go 1.24+, `net/http`, pgx/PostgreSQL 17, React 19, TypeScript 5.8, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-18-gis-house-profile-design.md`

## Global Constraints

- Do not use SOAP, credentials, certificates, or HTML parsing.
- Keep all undocumented GIS ЖКХ paths and response shapes inside `backend/integration/gishousing`.
- Use only the fixed configured GIS ЖКХ HTTPS origin; never accept an upstream URL from an HTTP request.
- Cache successful profiles for 24 hours by default and return stale cached data when refresh fails.
- Never substitute the existing demo passport when no published profile is available.
- Keep GIS ЖКХ management organizations separate from request-routing organizations.
- Automated tests must not call the live GIS ЖКХ service.

---

### Task 1: GIS ЖКХ HTTP adapter

**Files:**
- Create: `backend/integration/gishousing/client.go`
- Create: `backend/integration/gishousing/client_test.go`
- Modify: `backend/domain/models.go`

**Interfaces:**
- Consumes: GAR/FIAS house GUID string.
- Produces: `type GISRegistryClient interface { FetchHouse(context.Context, string) (domain.HouseProfile, error) }` and sentinel errors `domain.ErrHouseProfileNotFound`, `domain.ErrHouseProfileUnavailable`.

- [ ] **Step 1: Write failing adapter tests**

Cover the two-call flow with `httptest.Server`: the first GET returns one exact entry from `houseList`, the second GET returns a complete house card. Assert normalized cadastral number, nullable areas/counts/year, organization/contact, GIS GUID/type, and retained raw JSON. Add tests for empty matches, malformed JSON, non-2xx responses, and an oversized body.

```go
profile, err := NewClient(server.URL, time.Second, server.Client()).FetchHouse(ctx, fiasGUID)
if err != nil { t.Fatal(err) }
if profile.GISHouseGUID != gisGUID || value(profile.Floors) != 16 {
    t.Fatalf("unexpected profile: %+v", profile)
}
```

- [ ] **Step 2: Run adapter tests and verify RED**

Run: `cd backend && go test ./integration/gishousing -run TestClient -v`

Expected: FAIL because the package and client do not exist.

- [ ] **Step 3: Add profile domain types and minimal adapter**

Add `domain.HouseProfile` with nullable pointer fields, GIS identity, management text fields, `RawPayload json.RawMessage`, `FetchedAt`, and `Stale`. Implement a bounded JSON reader, exact FIAS GUID match, and detail fetch:

```go
type Client struct {
    BaseURL string
    HTTP    *http.Client
    Now     func() time.Time
}

func (c Client) FetchHouse(ctx context.Context, fiasGUID string) (domain.HouseProfile, error)
```

Use the public paths observed in the GIS ЖКХ frontend:

```text
/homemanagement/api/rest/services/houses/public/houses/searchByFiasHouseCodeList/{fiasGUID}?useReadOnlyDataSource=true
/homemanagement/api/rest/services/houses/public/{houseTypeCode}/{houseGuid}
```

Set `Accept: application/json; charset=utf-8` and a product-specific User-Agent. Limit each response to 2 MiB.

- [ ] **Step 4: Run adapter tests and all backend tests**

Run: `cd backend && go test ./integration/gishousing -v && go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/domain/models.go backend/integration/gishousing
git commit -m "feat: add public GIS house client"
```

### Task 2: PostgreSQL profile cache

**Files:**
- Create: `backend/repository/migrations/005_gis_house_profiles.sql`
- Create: `backend/repository/house_profile.go`
- Create: `backend/repository/house_profile_test.go`
- Modify: `backend/repository/postgres.go`

**Interfaces:**
- Consumes: `domain.HouseProfile` keyed by application house ID.
- Produces: `GetHouseProfile(context.Context, string) (domain.HouseProfile, error)`, `SaveHouseProfile(context.Context, domain.HouseProfile) error`, and `WithHouseProfileLock(context.Context, string, func(context.Context) error) error`.

- [ ] **Step 1: Write failing repository and migration tests**

Add tests following the existing repository SQL-builder/testing style. Verify missing rows become `domain.ErrNotFound`, every nullable field is scanned, an upsert replaces normalized and raw data, and migration version 5 is registered.

```go
if !strings.Contains(gisHouseProfilesMigration, "CREATE TABLE gis_house_profiles") {
    t.Fatal("profile table migration missing")
}
```

- [ ] **Step 2: Run repository tests and verify RED**

Run: `cd backend && go test ./repository -run 'HouseProfile|Migration' -v`

Expected: FAIL because migration 5 and cache methods do not exist.

- [ ] **Step 3: Add migration and repository methods**

Create `gis_house_profiles` with a unique `house_id` foreign key, GIS GUID/type, nullable normalized fields, `raw_payload jsonb NOT NULL`, and timestamps. Register embedded migration version 5. Use `pg_advisory_xact_lock` on a stable hash of house ID while refreshing.

- [ ] **Step 4: Run repository and backend tests**

Run: `cd backend && go test ./repository -v && go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/repository
git commit -m "feat: cache GIS house profiles"
```

### Task 3: Profile refresh orchestration

**Files:**
- Modify: `backend/service/house.go`
- Modify: `backend/service/house_test.go`
- Modify: `backend/domain/models.go`

**Interfaces:**
- Consumes: `HouseRepository`, `HouseProfileRepository`, `GISRegistryClient`, clock, and TTL.
- Produces: `HouseService.Resolve(context.Context, string) (domain.House, error)` returning address identity plus normalized profile/cache metadata.

- [ ] **Step 1: Add failing service tests**

Test these independent behaviors with real service logic and small fakes:

```text
fresh cache -> no upstream call
missing cache -> fetch and save
stale cache + refresh success -> new profile
stale cache + refresh failure -> old profile with stale=true
missing cache + no match -> ErrHouseProfileNotFound
missing cache + transport failure -> ErrHouseProfileUnavailable
```

Also assert GAR validation still happens before cache/upstream access.

- [ ] **Step 2: Run service tests and verify RED**

Run: `cd backend && go test ./service -run 'ResolveHouse|HouseProfile' -v`

Expected: FAIL because profile dependencies and refresh behavior are absent.

- [ ] **Step 3: Implement minimal cache/refresh service**

Extend `HouseService` with:

```go
Profiles HouseProfileRepository
Registry GISRegistryClient
ProfileTTL time.Duration
Now func() time.Time
```

Keep `ResolveHouse` responsible only for GAR-to-application-house identity. Compose the returned `domain.House` from that canonical address plus profile fields. On a stale fallback, preserve the original `FetchedAt` and set `Stale=true`.

- [ ] **Step 4: Run service and backend tests**

Run: `cd backend && go test ./service -v && go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/domain/models.go backend/service/house.go backend/service/house_test.go
git commit -m "feat: refresh cached house profiles"
```

### Task 4: HTTP contract, configuration, and server wiring

**Files:**
- Modify: `backend/config/config.go`
- Create or modify: `backend/config/config_test.go`
- Modify: `backend/controller/http.go`
- Modify: `backend/controller/address_test.go`
- Modify: `backend/controller/admin.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `.env.example`
- Modify: `docker-compose.yml`
- Modify: `docker-compose.prod.yml`

**Interfaces:**
- Consumes: `GIS_HOUSING_BASE_URL`, `GIS_HOUSING_TIMEOUT`, and `GIS_HOUSE_CACHE_TTL`.
- Produces: extended `POST /api/houses/resolve`; removes fixed `GET /api/house`.

- [ ] **Step 1: Write failing controller/config tests**

Assert defaults are the official HTTPS origin, 8-second request timeout, and 24-hour cache TTL; invalid duration values fail configuration loading rather than silently changing behavior. Assert resolve serializes nullable fields plus `dataSource`, `dataUpdatedAt`, and `stale`; profile-not-found maps to 404 and unavailable maps to 503. Assert `/api/house` is not registered.

- [ ] **Step 2: Run targeted tests and verify RED**

Run: `cd backend && go test ./config ./controller -run 'GIS|ResolveHouse|LegacyHouse' -v`

Expected: FAIL on missing configuration and response behavior.

- [ ] **Step 3: Implement configuration and wiring**

Construct `gishousing.Client` with a dedicated `http.Client`, inject repository/profile dependencies into `HouseService`, remove the fixed route and `MyHouse` repository method, and document/pass environment values through both Compose files.

- [ ] **Step 4: Run backend verification**

Run: `cd backend && go fmt ./... && go test ./... && go vet ./...`

Expected: PASS with no vet diagnostics.

- [ ] **Step 5: Commit**

```bash
git add .env.example docker-compose.yml docker-compose.prod.yml backend
git commit -m "feat: expose selected house profiles"
```

### Task 5: Resident house selection UI

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/pages/MyHouse.tsx`
- Modify: `frontend/src/styles.css`
- Create: `frontend/tests/my-house-selection.spec.ts`
- Modify: `frontend/tests/base-path.spec.ts`

**Interfaces:**
- Consumes: `POST /api/houses/resolve` and existing `AddressAutocomplete`.
- Produces: browser-local key `tvoy-dom:selected-house-object-id` and complete empty/loading/profile/stale/error UI states.

- [ ] **Step 1: Add failing Playwright tests**

Route address search and resolve responses. Cover first-visit selection, saved ID reload, change/clear, null fields as `Нет данных`, stale warning, retryable error, and assert no request reaches `/api/house`.

```ts
await page.goto("/max/house");
await expect(page.getByRole("combobox", { name: "Адрес дома" })).toBeVisible();
await page.getByRole("option", { name: house.fullAddress }).click();
await expect(page.getByRole("heading", { name: house.fullAddress })).toBeVisible();
expect(await page.evaluate(() => localStorage.getItem("tvoy-dom:selected-house-object-id"))).toBe(house.objectId);
```

- [ ] **Step 2: Run the new browser test and verify RED**

Run with the existing Playwright server setup: `cd frontend && pnpm test:e2e -- my-house-selection.spec.ts`

Expected: FAIL because `MyHouse` still calls `/api/house` and has no selector.

- [ ] **Step 3: Implement minimal frontend flow**

Add `houseApi.resolve(objectId, signal)`, make nullable profile fields explicit in `House`, restore the local object ID on mount, resolve after selection, and render `Нет данных` for null values. Include buttons to change/clear selection and a warning for `stale=true`.

- [ ] **Step 4: Run frontend tests and build**

Run: `cd frontend && pnpm test:e2e -- my-house-selection.spec.ts && pnpm build`

Expected: PASS.

- [ ] **Step 5: Run all frontend browser tests**

Run: `cd frontend && pnpm test:e2e`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend
git commit -m "feat: let residents select their house"
```

### Task 6: Documentation and full verification

**Files:**
- Modify: `README.md`
- Modify: `scripts/smoke.py`

**Interfaces:**
- Consumes: final backend/frontend behavior.
- Produces: deployment instructions and a smoke check that never relies on fabricated house data.

- [ ] **Step 1: Update documentation and smoke expectations**

Document the public JSON integration, cache/fallback semantics, new environment variables, browser-local selection, and the fact that `MOCK_STATUS_ENABLED` controls only status transitions. Replace the fixed `/api/house` smoke assertion with a controlled resolve/profile assertion or omit the external-dependent check when no fake upstream is configured.

- [ ] **Step 2: Run formatting and repository checks**

Run:

```bash
cd backend && go fmt ./... && go test ./... && go vet ./...
cd ../frontend && pnpm build && pnpm test:e2e
cd .. && docker compose config >/dev/null && docker compose -f docker-compose.prod.yml config >/dev/null
git diff --check
```

Expected: every command exits 0.

- [ ] **Step 3: Review against the specification**

Confirm every spec section has an implementation: browser-only selection, no legacy demo endpoint, isolated JSON adapter, PostgreSQL cache, 24-hour freshness, advisory locking, stale fallback, nullable fields, fixed upstream origin, logs, tests, and documentation.

- [ ] **Step 4: Commit**

```bash
git add README.md scripts/smoke.py
git commit -m "docs: explain GIS house profiles"
```
