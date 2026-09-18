# Enriched House Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract, cache, and display the maximum reliable house and management facts already exposed by the public GIS Housing detail and square-summary endpoints, while preserving the existing API fields and graceful stale-cache behavior.

**Architecture:** Keep `FetchHouse` as the service boundary and turn its internals into a three-call adapter: mandatory lookup, mandatory detail, and optional square summary. Normalize vendor JSON into typed domain objects, persist the typed values plus both raw successful payloads, expose nested `characteristics`, `management`, and `dataSources` objects alongside the legacy flat API fields, and render those objects on the existing My House page. Fix the GAR house display projection independently so the selected address includes its verified structure suffix.

**Tech Stack:** Go 1.23, `net/http`, PostgreSQL/pgx, embedded SQL migrations, React 19, TypeScript, Vite, Playwright, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-18-enriched-house-profile-design.md`

## Global Constraints

- Follow test-driven development for every behavior change: add one focused failing test, observe the intended failure, implement the minimum change, then rerun the focused test.
- Never fabricate missing GIS Housing facts. JSON `null`, absent properties, empty strings, and failed optional square-summary calls stay nullable.
- The lookup and detail requests are mandatory. The square-summary request is best effort and must not turn a valid detail card into an error.
- Keep the existing `FetchHouse(context.Context, string) (domain.HouseProfile, error)` interface and all existing top-level response fields during this migration.
- Keep raw upstream JSON server-side. `RawPayload` remains the detail payload; `SquarePayload` stores only a successful square response.
- Preserve the existing 2 MiB response-body limit for every GIS Housing call.
- Do not add suppliers, capital repair, equipment, tariffs, protected endpoints, or routing in this change.
- Do not call live GIS Housing in automated tests. Use reduced captured fixtures.
- Run `gofmt` on every changed Go file and preserve unrelated user changes.

## Responsibility Map

| Area | Files | Contract produced or consumed |
|---|---|---|
| Domain model | `backend/domain/models.go` | Typed house characteristics, management, and source metadata shared by integration, repository, service, and controller layers |
| GIS adapter | `backend/integration/gishousing/client.go` | Produces a normalized `domain.HouseProfile` from mandatory lookup/detail plus optional square summary |
| GIS fixtures/tests | `backend/integration/gishousing/client_test.go`, `backend/integration/gishousing/testdata/*.json` | Locks observed field names, numeric coercion, precedence, and optional-failure behavior |
| Persistence | `backend/repository/migrations/007_enriched_gis_house_profiles.sql`, `backend/repository/postgres.go`, `backend/repository/house_profile.go`, `backend/repository/house_profile_test.go`, `backend/repository/house_profile_integration_test.go` | Additive cache schema, exact select/scan/upsert order, and database round trips |
| Service mapping | `backend/service/house.go`, `backend/service/house_test.go` | Copies normalized profile data to the public house model without changing cache semantics |
| HTTP response | `backend/controller/http.go`, `backend/controller/address_test.go` | Preserves legacy flat fields and adds nested response objects |
| GAR display | `gar-init/internal/db/search.sql`, `gar-init/internal/db/store_test.go` | Emits verified `стр.` additions without duplicates |
| Frontend model/UI | `frontend/src/types.ts`, `frontend/src/pages/MyHouse.tsx`, `frontend/tests/my-house-selection.spec.ts`, `frontend/tests/base-path.spec.ts` | Consumes nested objects, renders all published facts and source availability |
| Documentation | `README.md`, `gar-init/README.md` if GAR projection is documented there | Describes enriched fields, optional-source behavior, and address suffix handling |

---

## Task 1: Define the normalized domain contract

**Files:**

- Modify: `backend/domain/models.go`
- Test: `backend/service/house_test.go`

**Interfaces:**

- Produce `domain.HouseCharacteristics`, `domain.HouseManagement`, and `domain.HouseDataSource`.
- Extend both `domain.HouseProfile` and `domain.House` with the structured values.
- Keep every current flat field so existing service/controller/frontend callers compile.

- [ ] **Step 1: Add a failing service test for structured profile propagation**

Append a focused test that constructs a populated profile and asserts that `applyProfile` keeps both nested and legacy values:

```go
func TestApplyProfileCopiesStructuredAndLegacyFields(t *testing.T) {
	updated := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	total := 6720.8
	shortName := `ГБУ "Жилищник района Арбат"`
	profile := domain.HouseProfile{
		Characteristics: domain.HouseCharacteristics{
			HouseType: pointer("Многоквартирный"),
			TotalArea: &total,
		},
		Management: domain.HouseManagement{ShortName: &shortName},
		FetchedAt: updated,
	}

	house := applyProfile(domain.House{}, profile)
	if house.Characteristics.HouseType == nil || *house.Characteristics.HouseType != "Многоквартирный" {
		t.Fatalf("characteristics not copied: %+v", house)
	}
	if house.TotalArea == nil || *house.TotalArea != total || house.Organization == nil || *house.Organization != shortName {
		t.Fatalf("legacy fields not derived: %+v", house)
	}
}
```

Use a package-local generic pointer helper in the test if none exists:

```go
func pointer[T any](value T) *T { return &value }
```

- [ ] **Step 2: Run the focused test and confirm RED**

Run:

```bash
cd backend && go test ./service -run TestApplyProfileCopiesStructuredAndLegacyFields -count=1
```

Expected: compile failure because the new domain types/fields do not exist.

- [ ] **Step 3: Add the domain types and fields**

Add these exact nullable contracts to `backend/domain/models.go`:

```go
type HouseCharacteristics struct {
	HouseTypeCode                         *string    `json:"houseTypeCode"`
	HouseType                             *string    `json:"houseType"`
	Status                                *string    `json:"status"`
	ProjectSeries                         *string    `json:"projectSeries"`
	Condition                             *string    `json:"condition"`
	LifecycleStage                        *string    `json:"lifecycleStage"`
	YearBuilt                             *int       `json:"yearBuilt"`
	OperationYear                         *int       `json:"operationYear"`
	ReconstructionYear                    *int       `json:"reconstructionYear"`
	DeteriorationPercent                  *float64   `json:"deteriorationPercent"`
	DeteriorationDate                     *time.Time `json:"deteriorationDate"`
	WallMaterial                          *string    `json:"wallMaterial"`
	EnergyEfficiency                      *string    `json:"energyEfficiency"`
	TotalArea                             *float64   `json:"totalArea"`
	LivingArea                            *float64   `json:"livingArea"`
	NonResidentialArea                    *float64   `json:"nonResidentialArea"`
	ResidentialPremises                   *int       `json:"residentialPremises"`
	ResidentialPremisesArea               *float64   `json:"residentialPremisesArea"`
	ResidentialPremisesWithRealty         *int       `json:"residentialPremisesWithRealty"`
	ResidentialPremisesWithRealtyArea     *float64   `json:"residentialPremisesWithRealtyArea"`
	NonResidentialPremises                *int       `json:"nonResidentialPremises"`
	NonResidentialPremisesArea            *float64   `json:"nonResidentialPremisesArea"`
	NonResidentialPremisesNotCommon       *int       `json:"nonResidentialPremisesNotCommon"`
	NonResidentialPremisesNotCommonArea   *float64   `json:"nonResidentialPremisesNotCommonArea"`
	Floors                                *int       `json:"floors"`
	Entrances                             *int       `json:"entrances"`
	OwnersOrShares                        *int       `json:"ownersOrShares"`
}

type HouseManagement struct {
	Method                   *string    `json:"method"`
	OrganizationGUID         *string    `json:"organizationGuid"`
	ShortName                *string    `json:"shortName"`
	FullName                 *string    `json:"fullName"`
	Address                  *string    `json:"address"`
	Phone                    *string    `json:"phone"`
	Website                  *string    `json:"website"`
	OrganizationType         *string    `json:"organizationType"`
	RegistryOrganizationGUID *string    `json:"registryOrganizationGuid"`
	INN                      *string    `json:"inn"`
	OGRN                     *string    `json:"ogrn"`
	Chief                    *string    `json:"chief"`
	ContractStart            *time.Time `json:"contractStart"`
	ContractEnd              *time.Time `json:"contractEnd"`
}

type HouseDataSource struct {
	Name      string    `json:"name"`
	Available bool      `json:"available"`
	UpdatedAt time.Time `json:"updatedAt"`
	Stale     bool      `json:"stale"`
}
```

Add `Characteristics HouseCharacteristics`, `Management HouseManagement`, and `DataSources []HouseDataSource` to `House` and `HouseProfile`. Add `SquarePayload json.RawMessage` and `SquareSummaryAvailable bool` to `HouseProfile`. Retain all old fields unchanged.

- [ ] **Step 4: Implement compatibility derivation in `applyProfile`**

Copy the structured objects, then derive current flat fields using the normalized object as the source of truth:

```go
house.Characteristics = profile.Characteristics
house.Management = profile.Management
house.DataSources = profile.DataSources
for index := range house.DataSources {
	house.DataSources[index].UpdatedAt = profile.FetchedAt
	house.DataSources[index].Stale = profile.Stale
}
house.CadastralNumber = profile.CadastralNumber
house.TotalArea = profile.Characteristics.TotalArea
house.LivingArea = profile.Characteristics.LivingArea
house.Floors = profile.Characteristics.Floors
house.Entrances = profile.Characteristics.Entrances
house.Apartments = profile.Characteristics.ResidentialPremises
house.YearBuilt = profile.Characteristics.YearBuilt
house.Organization = firstString(profile.Management.ShortName, profile.Management.FullName)
house.Manager = profile.Management.Chief
house.Contact = profile.Management.Phone
```

During compatibility with old cached rows, fall back to the old flat profile field when the corresponding structured field is nil. Do not overwrite non-nil legacy cached facts with nil structured values.

- [ ] **Step 5: Run and format**

```bash
gofmt -w backend/domain/models.go backend/service/house.go backend/service/house_test.go
cd backend && go test ./service -run 'TestApplyProfileCopiesStructuredAndLegacyFields|TestHouseProfile' -count=1
```

Expected: PASS, including existing fresh-cache and stale-fallback tests.

- [ ] **Step 6: Commit**

```bash
git add backend/domain/models.go backend/service/house.go backend/service/house_test.go
git commit -m "feat: define enriched house profile model"
```

---

## Task 2: Normalize the real GIS detail card and optional square summary

**Files:**

- Modify: `backend/integration/gishousing/client.go`
- Modify: `backend/integration/gishousing/client_test.go`
- Create: `backend/integration/gishousing/testdata/arbat_lookup.json`
- Create: `backend/integration/gishousing/testdata/arbat_detail.json`
- Create: `backend/integration/gishousing/testdata/arbat_square.json`

**Interfaces:**

- Consume the existing GIS public lookup and detail URLs.
- Add `GET /homemanagement/api/rest/services/houses/public/get-house-square-data/{gisHouseGuid}`.
- Continue producing `domain.HouseProfile`; callers do not change.

- [ ] **Step 1: Add reduced captured fixtures**

Store deterministic fixture JSON with the verified Arbat identifiers and representative number/string forms. The detail fixture must contain at least:

```json
{
  "guid": "105a84e6-01b5-4bbb-a26d-e5ec59bee321",
  "houseType": {"code": "1", "houseTypeName": "Многоквартирный"},
  "status": "APPROVED",
  "cadastreNumber": "77:01:0001046:1013",
  "totalSquare": "6720.8",
  "residentialSquare": 2405.3,
  "floorCount": null,
  "buildingYear": "1870",
  "operationYear": 1870,
  "reconstructionYear": "1870",
  "planSeries": "Индивидуальный проект",
  "houseCondition": {"houseCondition": "Исправный"},
  "lifeCycleStage": {"lifeCycleStage": "Эксплуатация"},
  "deterioration": "66",
  "deteriorationDate": "31.12.2009",
  "intWallMaterialList": "Стены кирпичные",
  "houseManagementType": {"code": "5", "houseManagementTypeName": "УО"},
  "managementContractDate": "22.04.2015",
  "endContractDate": null,
  "managementOrganization": {
    "guid": "management-guid",
    "shortName": "ГБУ Жилищник района Арбат",
    "fullName": "Государственное бюджетное учреждение города Москвы Жилищник района Арбат",
    "orgAddress": "121099, г. Москва, пер. Проточный, д. 9, стр. 1",
    "phone": "74952305787",
    "url": "https://arbatgbu.mos.ru/",
    "organizationType": "Управляющая организация",
    "registryOrganizationRootEntityGuid": "registry-guid",
    "ogrn": "5147746267906"
  },
  "chiefLastName": "Иванов",
  "chiefFirstName": "Иван",
  "chiefMiddleName": "Иванович"
}
```

The square fixture must contain all currently verified values:

```json
{
  "totalSquare": 6720.8,
  "residentialSquare": 2405.3,
  "residentialPremiseCount": 15,
  "residentialPremiseSquare": 2405.3,
  "residentialPremiseWithRelatedRealtyCount": 15,
  "nonresidentialSquare": 4023.6,
  "nonresidentialPremiseCount": 31,
  "nonresidentialPremiseSquare": 4176.4,
  "nonresidentialPremiseNotCommonCount": 29,
  "shareOwnersNumber": 61
}
```

- [ ] **Step 2: Replace the happy-path test with fixture-backed expectations**

Use `//go:embed testdata/arbat_*.json` and an `httptest.Server` that returns lookup, detail, then square data. Assert:

```go
if got := value(profile.Characteristics.HouseType); got != "Многоквартирный" {
	t.Fatalf("house type=%q", got)
}
if got := value(profile.Characteristics.ProjectSeries); got != "Индивидуальный проект" {
	t.Fatalf("project series=%q", got)
}
if got := value(profile.Characteristics.ResidentialPremises); got != 15 {
	t.Fatalf("residential premises=%d", got)
}
if got := value(profile.Characteristics.NonResidentialPremises); got != 31 {
	t.Fatalf("non-residential premises=%d", got)
}
if got := value(profile.Characteristics.OwnersOrShares); got != 61 {
	t.Fatalf("owners or shares=%d", got)
}
if profile.Characteristics.Floors != nil || profile.Characteristics.Entrances != nil {
	t.Fatalf("unpublished values must stay nil: %+v", profile.Characteristics)
}
if !profile.SquareSummaryAvailable || len(profile.SquarePayload) == 0 {
	t.Fatalf("square source not recorded: %+v", profile)
}
if got := value(profile.Management.OGRN); got != "5147746267906" {
	t.Fatalf("OGRN=%q", got)
}
```

Also assert that legacy `Apartments` is 15, legacy `YearBuilt` is 1870, and `RawPayload` contains the detail card rather than the lookup or square response.

- [ ] **Step 3: Add table tests for floor/count fallbacks and optional failures**

Add these behaviors as separate subtests:

1. `floorCount` wins; `floorCountMax` remains a legacy fallback.
2. Square `residentialPremiseCount` wins over detail `residentialPremiseActualCount`, then `residentialPremiseConfirmedCount`, then legacy `residentialPremiseCount`.
3. Square timeout, HTTP 502, malformed JSON, and body larger than 2 MiB each return a successful detail profile with `SquareSummaryAvailable == false` and `SquarePayload == nil`.

For timeout, use a custom round tripper that returns `context.DeadlineExceeded` only for `get-house-square-data`. For the other cases route by URL path in `httptest.Server`.

- [ ] **Step 4: Run the adapter tests and confirm RED**

```bash
cd backend && go test ./integration/gishousing -run 'TestClientFetchHouse|TestSquareSummary|TestFloorAndPremiseFallbacks' -count=1
```

Expected: failures because current DTOs ignore the new fields and `FetchHouse` does not request square data.

- [ ] **Step 5: Split vendor DTOs by endpoint**

Use endpoint-specific private structs rather than one catch-all struct. Keep both current and legacy variants explicitly:

```go
type referenceDTO struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

type houseDetailDTO struct {
	GUID                             string         `json:"guid"`
	CadastreNumber                  *string        `json:"cadastreNumber"`
	TotalSquare                      *flexibleFloat `json:"totalSquare"`
	ResidentialSquare                *flexibleFloat `json:"residentialSquare"`
	FloorCount                       *flexibleInt   `json:"floorCount"`
	FloorCountMax                    *flexibleInt   `json:"floorCountMax"`
	EntranceCount                    *flexibleInt   `json:"entranceCount"`
	ResidentialPremiseCount          *flexibleInt   `json:"residentialPremiseCount"`
	ResidentialPremiseActualCount    *flexibleInt   `json:"residentialPremiseActualCount"`
	ResidentialPremiseConfirmedCount *flexibleInt   `json:"residentialPremiseConfirmedCount"`
	BuildingYear                     *flexibleInt   `json:"buildingYear"`
	OperationYear                    *flexibleInt   `json:"operationYear"`
	ReconstructionYear               *flexibleInt   `json:"reconstructionYear"`
	PlanSeries                       string         `json:"planSeries"`
	Status                           string         `json:"status"`
	Deterioration                    *flexibleFloat `json:"deterioration"`
	DeteriorationDate                flexibleDate   `json:"deteriorationDate"`
	WallMaterial                     string         `json:"intWallMaterialList"`
	EnergyEfficiency                 referenceDTO   `json:"houseEnergyEfficiency"`
	ManagementContractDate           flexibleDate   `json:"managementContractDate"`
	EndContractDate                  flexibleDate   `json:"endContractDate"`
	ChiefLastName                    string         `json:"chiefLastName"`
	ChiefFirstName                   string         `json:"chiefFirstName"`
	ChiefMiddleName                  string         `json:"chiefMiddleName"`
	HouseType struct {
		Code          string `json:"code"`
		HouseTypeName string `json:"houseTypeName"`
	} `json:"houseType"`
	HouseCondition struct {
		HouseCondition string `json:"houseCondition"`
	} `json:"houseCondition"`
	LifecycleStage struct {
		LifecycleStage string `json:"lifeCycleStage"`
	} `json:"lifeCycleStage"`
	ManagementType struct {
		Name string `json:"houseManagementTypeName"`
	} `json:"houseManagementType"`
	ManagementOrganization struct {
		GUID                           string `json:"guid"`
		ShortName                      string `json:"shortName"`
		FullName                       string `json:"fullName"`
		Address                        string `json:"orgAddress"`
		Phone                          string `json:"phone"`
		Website                        string `json:"url"`
		OrganizationType               string `json:"organizationType"`
		RegistryOrganizationRootGUID   string `json:"registryOrganizationRootEntityGuid"`
		INN                            string `json:"inn"`
		OGRN                           string `json:"ogrn"`
	} `json:"managementOrganization"`
}

type squareSummaryDTO struct {
	TotalSquare                                  *flexibleFloat `json:"totalSquare"`
	ResidentialSquare                            *flexibleFloat `json:"residentialSquare"`
	NonResidentialSquare                         *flexibleFloat `json:"nonresidentialSquare"`
	ResidentialPremiseCount                      *flexibleInt   `json:"residentialPremiseCount"`
	ResidentialPremiseSquare                     *flexibleFloat `json:"residentialPremiseSquare"`
	ResidentialPremiseWithRelatedRealtyCount     *flexibleInt   `json:"residentialPremiseWithRelatedRealtyCount"`
	ResidentialPremiseWithRelatedRealtySquare    *flexibleFloat `json:"residentialPremiseWithRelatedRealtySquare"`
	NonResidentialPremiseCount                   *flexibleInt   `json:"nonresidentialPremiseCount"`
	NonResidentialPremiseSquare                  *flexibleFloat `json:"nonresidentialPremiseSquare"`
	NonResidentialPremiseNotCommonCount          *flexibleInt   `json:"nonresidentialPremiseNotCommonCount"`
	NonResidentialPremiseNotCommonSquare         *flexibleFloat `json:"nonresidentialPremiseNotCommonSquare"`
	ShareOwnersNumber                            *flexibleInt   `json:"shareOwnersNumber"`
}
```

Implement `flexibleDate` for `DD.MM.YYYY`, `YYYY-MM-DD`, and RFC3339, returning nil for JSON null/empty string and an unmarshalling error for a non-empty invalid date. Use small helpers `firstInt`, `firstFloat`, `firstText`, and `joinPublished` so precedence is visible and tested.

- [ ] **Step 6: Build the profile from detail, then merge the square summary**

After a successful detail fetch:

```go
profile := profileFromDetail(detail, rawDetail, c.now().UTC())
squareURL := c.BaseURL + "/homemanagement/api/rest/services/houses/public/get-house-square-data/" + url.PathEscape(profile.GISHouseGUID)
var square squareSummaryDTO
if rawSquare, squareErr := c.getJSON(ctx, squareURL, &square); squareErr == nil {
	mergeSquareSummary(&profile, square)
	profile.SquarePayload = rawSquare
	profile.SquareSummaryAvailable = true
}
profile.DataSources = []domain.HouseDataSource{
	{Name: "ГИС ЖКХ · карточка дома", Available: true, UpdatedAt: profile.FetchedAt},
	{Name: "ГИС ЖКХ · сводка помещений", Available: profile.SquareSummaryAvailable, UpdatedAt: profile.FetchedAt},
}
return profile, nil
```

`mergeSquareSummary` replaces only facts actually present in the square response. It must not clear a detail fallback. Recompute the legacy fields from the final structured objects before returning.

- [ ] **Step 7: Run all adapter tests and format**

```bash
gofmt -w backend/integration/gishousing/client.go backend/integration/gishousing/client_test.go
cd backend && go test ./integration/gishousing -count=1
```

Expected: PASS for the new fixtures, all optional failures, and the original mandatory error matrix.

- [ ] **Step 8: Commit**

```bash
git add backend/integration/gishousing backend/domain/models.go
git commit -m "feat: extract enriched GIS house data"
```

---

## Task 3: Persist all normalized fields and both raw payloads

**Files:**

- Create: `backend/repository/migrations/007_enriched_gis_house_profiles.sql`
- Modify: `backend/repository/postgres.go`
- Modify: `backend/repository/house_profile.go`
- Modify: `backend/repository/house_profile_test.go`
- Create: `backend/repository/house_profile_integration_test.go`

**Interfaces:**

- Consume and produce the expanded `domain.HouseProfile` without changing repository method signatures.
- Migration version is exactly `7` and remains additive for existing rows.

- [ ] **Step 1: Extend repository tests before SQL**

Add a test that requires version 7 to be embedded and checks every new column in the migration, select, and upsert strings. Use one canonical list:

```go
var enrichedProfileColumns = []string{
	"house_type_name", "house_status", "project_series", "house_condition",
	"lifecycle_stage", "operation_year", "reconstruction_year",
	"deterioration_percent", "deterioration_date", "wall_material",
	"energy_efficiency", "non_residential_area", "residential_premises",
	"residential_premises_area", "residential_premises_with_realty",
	"residential_premises_with_realty_area", "non_residential_premises",
	"non_residential_premises_area", "non_residential_premises_not_common",
	"non_residential_premises_not_common_area", "owners_or_shares",
	"management_method", "management_organization_guid", "management_short_name",
	"management_full_name", "management_address", "management_phone",
	"management_website", "management_organization_type",
	"management_registry_organization_guid", "management_inn", "management_ogrn",
	"management_chief", "management_contract_start", "management_contract_end",
	"square_payload", "square_summary_available",
}
```

Also assert `strings.Contains(postgres source behavior, version=7)` indirectly by exposing the embedded `enrichedGISHouseProfilesMigration` variable to the package test and checking it is non-empty.

- [ ] **Step 2: Run repository tests and confirm RED**

```bash
cd backend && go test ./repository -run 'TestEnrichedHouseProfile|TestHouseProfileQueries' -count=1
```

Expected: compile or assertion failure because migration 007 and its columns do not exist.

- [ ] **Step 3: Add migration 007**

Create an idempotent additive migration using `ADD COLUMN IF NOT EXISTS`. Use these PostgreSQL types:

```sql
ALTER TABLE gis_house_profiles
    ADD COLUMN IF NOT EXISTS house_type_name text,
    ADD COLUMN IF NOT EXISTS house_status text,
    ADD COLUMN IF NOT EXISTS project_series text,
    ADD COLUMN IF NOT EXISTS house_condition text,
    ADD COLUMN IF NOT EXISTS lifecycle_stage text,
    ADD COLUMN IF NOT EXISTS operation_year integer,
    ADD COLUMN IF NOT EXISTS reconstruction_year integer,
    ADD COLUMN IF NOT EXISTS deterioration_percent double precision,
    ADD COLUMN IF NOT EXISTS deterioration_date date,
    ADD COLUMN IF NOT EXISTS wall_material text,
    ADD COLUMN IF NOT EXISTS energy_efficiency text,
    ADD COLUMN IF NOT EXISTS non_residential_area double precision,
    ADD COLUMN IF NOT EXISTS residential_premises integer,
    ADD COLUMN IF NOT EXISTS residential_premises_area double precision,
    ADD COLUMN IF NOT EXISTS residential_premises_with_realty integer,
    ADD COLUMN IF NOT EXISTS residential_premises_with_realty_area double precision,
    ADD COLUMN IF NOT EXISTS non_residential_premises integer,
    ADD COLUMN IF NOT EXISTS non_residential_premises_area double precision,
    ADD COLUMN IF NOT EXISTS non_residential_premises_not_common integer,
    ADD COLUMN IF NOT EXISTS non_residential_premises_not_common_area double precision,
    ADD COLUMN IF NOT EXISTS owners_or_shares integer,
    ADD COLUMN IF NOT EXISTS management_method text,
    ADD COLUMN IF NOT EXISTS management_organization_guid text,
    ADD COLUMN IF NOT EXISTS management_short_name text,
    ADD COLUMN IF NOT EXISTS management_full_name text,
    ADD COLUMN IF NOT EXISTS management_address text,
    ADD COLUMN IF NOT EXISTS management_phone text,
    ADD COLUMN IF NOT EXISTS management_website text,
    ADD COLUMN IF NOT EXISTS management_organization_type text,
    ADD COLUMN IF NOT EXISTS management_registry_organization_guid text,
    ADD COLUMN IF NOT EXISTS management_inn text,
    ADD COLUMN IF NOT EXISTS management_ogrn text,
    ADD COLUMN IF NOT EXISTS management_chief text,
    ADD COLUMN IF NOT EXISTS management_contract_start date,
    ADD COLUMN IF NOT EXISTS management_contract_end date,
    ADD COLUMN IF NOT EXISTS square_payload jsonb,
    ADD COLUMN IF NOT EXISTS square_summary_available boolean NOT NULL DEFAULT false;
```

- [ ] **Step 4: Embed and apply migration version 7**

In `postgres.go`, add:

```go
//go:embed migrations/007_enriched_gis_house_profiles.sql
var enrichedGISHouseProfilesMigration string
```

After version 6, use the same transaction pattern to query `version=7`, execute the migration, and insert `schema_migrations VALUES(7)`.

- [ ] **Step 5: Expand select/scan/upsert in one canonical order**

Place identity and legacy columns first, structured characteristic columns next, management columns next, and payload/metadata columns last. Keep `selectHouseProfile`, `Scan`, `upsertHouseProfile`, and `Exec` argument order identical. Explicitly map every column into `profile.Characteristics.*` or `profile.Management.*`.

After scanning, derive `profile.DataSources` from `FetchedAt`, `Stale`, and `SquareSummaryAvailable`; also backfill structured fields from legacy columns for pre-refresh rows. Before saving, derive legacy columns from structured values so mixed-version callers stay safe.

- [ ] **Step 6: Add populated and partial database round-trip tests**

Create a repository integration test guarded by `HOUSE_PROFILE_TEST_DATABASE_URL`. Its helper must:

1. connect to the supplied PostgreSQL server;
2. create a uniquely named schema;
3. configure a pool with `search_path` set to that schema;
4. create a minimal `houses(id bigint primary key)` table;
5. execute migrations 005 and 007;
6. drop the schema in `t.Cleanup`.

The populated test inserts `houses(id) VALUES(42)`, calls `SaveHouseProfile` with every new field populated and both payloads valid JSON, calls `GetHouseProfile`, and compares every characteristics/management field, `SquareSummaryAvailable`, both raw payloads, and `FetchedAt`.

The partial test saves a profile whose optional fields and `SquarePayload` are nil, reloads it, and asserts those values remain nil rather than zero/empty values. It also asserts the square source is returned with `Available == false`.

Do not silently skip when the variable is present but setup fails:

```go
databaseURL := os.Getenv("HOUSE_PROFILE_TEST_DATABASE_URL")
if databaseURL == "" {
	t.Skip("HOUSE_PROFILE_TEST_DATABASE_URL is not set")
}
```

- [ ] **Step 7: Run repository and backend package tests**

```bash
gofmt -w backend/repository/postgres.go backend/repository/house_profile.go backend/repository/house_profile_test.go
cd backend && go test ./repository ./service ./integration/gishousing -count=1
```

Expected: PASS with nullable values preserved and no scan/argument count mismatch. Without the integration database, the two explicit round-trip cases report the documented environment skip.

- [ ] **Step 8: Commit**

```bash
git add backend/repository
git commit -m "feat: persist enriched house profiles"
```

---

## Task 4: Expose structured response objects without breaking clients

**Files:**

- Modify: `backend/controller/http.go`
- Modify: `backend/controller/address_test.go`
- Modify: `backend/service/house_test.go`

**Interfaces:**

- `POST /api/houses/resolve` and `GET /api/houses/{id}` keep all current top-level keys.
- Both routes additionally return `characteristics`, `management`, and `dataSources`.
- Raw payloads are never serialized.

- [ ] **Step 1: Add a failing controller contract test**

Expand `TestResolveHouseAcceptsObjectID` with populated nested domain objects. Decode into a typed local response or `map[string]any`, then assert:

```go
characteristics := body["characteristics"].(map[string]any)
management := body["management"].(map[string]any)
sources := body["dataSources"].([]any)
if characteristics["residentialPremises"] != float64(15) {
	t.Fatalf("unexpected characteristics: %v", characteristics)
}
if management["ogrn"] != "5147746267906" {
	t.Fatalf("unexpected management: %v", management)
}
if len(sources) != 2 {
	t.Fatalf("unexpected sources: %v", sources)
}
if body["apartments"] != float64(15) || body["organization"] != "ГБУ Жилищник района Арбат" {
	t.Fatalf("legacy fields changed: %v", body)
}
if _, exposed := body["rawPayload"]; exposed { t.Fatal("raw payload must stay private") }
```

- [ ] **Step 2: Run and confirm RED**

```bash
cd backend && go test ./controller -run TestResolveHouseAcceptsObjectID -count=1
```

Expected: nested fields are absent.

- [ ] **Step 3: Extend `houseResponse`**

Add only the structured public objects:

```go
"characteristics": h.Characteristics,
"management":      h.Management,
"dataSources":     h.DataSources,
```

Keep all existing keys unchanged. Do not return `HouseProfile`, `RawPayload`, or `SquarePayload` directly.

- [ ] **Step 4: Verify compatibility and service cache behavior**

```bash
gofmt -w backend/controller/http.go backend/controller/address_test.go backend/service/house_test.go
cd backend && go test ./controller ./service -count=1
```

Expected: PASS for the new nested response, all legacy fields, fresh-cache skip, and stale fallback.

- [ ] **Step 5: Commit**

```bash
git add backend/controller backend/service
git commit -m "feat: expose structured house profile API"
```

---

## Task 5: Preserve the verified GAR structure suffix

**Files:**

- Modify: `gar-init/internal/db/search.sql`
- Modify: `gar-init/internal/db/store_test.go`

**Interfaces:**

- Object `67020241` from the demo fixture must render as `г. Москва, ул. Арбат, д. 4, стр. 1`.
- Only fixture-verified `add_type = 2` is labeled `стр.` in this change.
- A structure already emitted from `struc_num` must not be repeated from `add_num1` or `add_num2`.

- [ ] **Step 1: Add the exact Arbat assertion**

After `SeedDemo` and `Finalize`, query the fixture object directly:

```go
var arbat string
if err := store.pool.QueryRow(context.Background(), `
	SELECT full_address FROM gar_search_addresses
	WHERE object_kind='house' AND object_id=67020241
`).Scan(&arbat); err != nil {
	t.Fatal(err)
}
if arbat != "г. Москва, ул. Арбат, д. 4, стр. 1" {
	t.Fatalf("unexpected Arbat address %q", arbat)
}
if strings.Count(arbat, "стр. 1") != 1 {
	t.Fatalf("duplicate structure suffix in %q", arbat)
}
```

- [ ] **Step 2: Run and confirm RED**

```bash
cd gar-init && go test ./internal/db -run TestSeedDemoAndFinalizeBuildSearchEntries -count=1
```

Expected with `GAR_TEST_DATABASE_URL` configured: address is missing `стр. 1`. If the environment variable is absent, record the skip and also add a pure string assertion test against `searchSQL` so the code path is still checked in the default suite.

- [ ] **Step 3: Extend the house display expression**

Append verified additions to the current `concat_ws`, suppressing duplicates:

```sql
CASE
  WHEN add_type1 = 2 AND nullif(add_num1, '') IS NOT NULL
       AND add_num1 IS DISTINCT FROM struc_num
    THEN 'стр. ' || add_num1
END,
CASE
  WHEN add_type2 = 2 AND nullif(add_num2, '') IS NOT NULL
       AND add_num2 IS DISTINCT FROM struc_num
       AND add_num2 IS DISTINCT FROM add_num1
    THEN 'стр. ' || add_num2
END
```

Do not guess labels for other type codes.

- [ ] **Step 4: Run GAR tests**

```bash
gofmt -w gar-init/internal/db/store_test.go
cd gar-init && go test ./... -count=1
```

Expected: all unit tests PASS; the database-backed assertion either PASSes with the configured test database or reports the existing explicit skip.

- [ ] **Step 5: Commit**

```bash
git add gar-init/internal/db/search.sql gar-init/internal/db/store_test.go
git commit -m "fix: include GAR structure in house addresses"
```

---

## Task 6: Render the enriched passport and management sections

**Files:**

- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/pages/MyHouse.tsx`
- Modify: `frontend/tests/my-house-selection.spec.ts`
- Modify: `frontend/tests/base-path.spec.ts`

**Interfaces:**

- Consume the nested API objects from Task 4.
- Preserve the existing address picker, retry, stale warning, create-request link, and legacy route behavior.
- Use the exact missing-value copy `Не опубликовано в ГИС ЖКХ`.

- [ ] **Step 1: Expand the Playwright fixture and assertions**

Add nested fixture objects matching the TypeScript contracts. In the primary test, assert published values and source state:

```ts
await expect(page.getByRole("heading", { name: "Паспорт дома" })).toBeVisible();
await expect(page.getByText("Индивидуальный проект")).toBeVisible();
await expect(page.getByText("15").first()).toBeVisible();
await expect(page.getByRole("heading", { name: /Жилищник района Арбат/ })).toBeVisible();
await expect(page.getByRole("link", { name: "arbatgbu.mos.ru" })).toHaveAttribute("href", "https://arbatgbu.mos.ru/");
await expect(page.getByText("ГИС ЖКХ · сводка помещений")).toBeVisible();
```

Add a second test with null facts and `available: false` for the square source. Assert `Не опубликовано в ГИС ЖКХ` and `недоступен при обновлении`, while the house page and management section remain visible.

- [ ] **Step 2: Run the focused browser tests and confirm RED**

```bash
cd frontend && pnpm exec playwright test tests/my-house-selection.spec.ts tests/base-path.spec.ts
```

Expected: new headings, facts, links, and source availability are missing.

- [ ] **Step 3: Add exact TypeScript response types**

Define `HouseCharacteristics`, `HouseManagement`, and `HouseDataSource` with every field from the Go JSON tags and nullable values. Extend `House`; keep the nested objects optional in TypeScript for a safe rolling deployment, while the updated backend always returns them:

```ts
export interface HouseDataSource {
  name: string;
  available: boolean;
  updatedAt: string;
  stale: boolean;
}

export interface House {
  // existing fields stay unchanged
  characteristics?: HouseCharacteristics;
  management?: HouseManagement;
  dataSources?: HouseDataSource[];
}
```

- [ ] **Step 4: Replace generic missing copy and render the two panels**

Use dedicated safe formatters:

```tsx
const missing = "Не опубликовано в ГИС ЖКХ";
const text = (value: string | number | null | undefined) => value ?? missing;
const area = (value: number | null | undefined) =>
  value == null ? missing : `${new Intl.NumberFormat("ru-RU").format(value)} м²`;
const percent = (value: number | null | undefined) =>
  value == null ? missing : `${new Intl.NumberFormat("ru-RU").format(value)} %`;
```

Render all characteristic fields in logical groups: identity/type/status, years/project/condition, areas/premise counts, floors/entrances, wear/energy/walls. Render management method, names, IDs, address, chief, contract dates, phone, and website.

For links:

```tsx
const website = house.management.website;
const safeWebsite = website && /^https?:\/\//i.test(website) ? website : null;
```

Use `rel="noreferrer"` and `target="_blank"` for the website. Render phone as `href={`tel:${phone.replace(/[^+\d]/g, "")}`}` only when non-empty. Otherwise render the missing copy.

Render each data source with its availability label and use the common update time. Keep the stale warning separate from square availability.

- [ ] **Step 5: Keep legacy response fixtures valid during rollout**

Update `base-path.spec.ts` to include nested objects. In the component, defensively fall back to legacy flat fields when nested objects are absent during a rolling deployment:

```tsx
const characteristics = house.characteristics ?? {
  totalArea: house.totalArea,
  livingArea: house.livingArea,
  floors: house.floors,
  entrances: house.entrances,
  residentialPremises: house.apartments,
  yearBuilt: house.yearBuilt,
};
```

Keep this fallback local and temporary; do not reintroduce guessed values.

- [ ] **Step 6: Run frontend verification**

```bash
cd frontend && pnpm exec tsc --noEmit
cd frontend && pnpm exec playwright test tests/my-house-selection.spec.ts tests/base-path.spec.ts
cd frontend && pnpm build
```

Expected: typecheck, targeted browser tests, and production build PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src frontend/tests
git commit -m "feat: show enriched house passport"
```

---

## Task 7: Document and verify the complete change

**Files:**

- Modify: `README.md`
- Modify: `gar-init/README.md` only if its current GAR search documentation needs the suffix detail
- Review: all files changed in Tasks 1–6

- [ ] **Step 1: Update user-facing technical documentation**

Document:

- lookup/detail are mandatory and square summary is optional;
- the public API now has `characteristics`, `management`, and `dataSources` plus compatibility fields;
- cache TTL/stale behavior is unchanged;
- raw payloads stay in PostgreSQL and are not sent to browsers;
- the GAR projection includes the verified `стр.` addition;
- suppliers, capital repair, equipment, tariffs, and routing are deferred.

- [ ] **Step 2: Run formatting and static checks**

```bash
gofmt -w backend/domain/models.go backend/integration/gishousing/client.go backend/integration/gishousing/client_test.go backend/repository/postgres.go backend/repository/house_profile.go backend/repository/house_profile_test.go backend/service/house.go backend/service/house_test.go backend/controller/http.go backend/controller/address_test.go gar-init/internal/db/store_test.go
cd backend && go vet ./...
cd gar-init && go vet ./...
cd frontend && pnpm exec tsc --noEmit
```

Expected: all commands exit 0.

- [ ] **Step 3: Run complete automated suites**

```bash
cd backend && go test ./... -count=1
cd gar-init && go test ./... -count=1
cd frontend && pnpm exec playwright test
cd frontend && pnpm build
```

Expected: all available tests PASS. A GAR database test may retain its explicit `GAR_TEST_DATABASE_URL is not set` skip; no other new test may be skipped.

- [ ] **Step 4: Validate deployment configuration and migration order**

```bash
docker compose config --quiet
rg -n "version=7|VALUES\(7\)|007_enriched" backend/repository
```

Expected: Compose configuration exits 0 and the search shows embed, version check, migration execution, and version insert.

- [ ] **Step 5: Inspect the final diff for contract and safety regressions**

```bash
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
rg -n "rawPayload|squarePayload" backend frontend gar-init README.md
```

Expected: no whitespace errors; raw payload names appear only in server-side persistence/integration code and tests, never in frontend public types or controller response construction.

- [ ] **Step 6: Optional read-only live comparison**

If network access is available, issue read-only requests for the known Arbat FIAS/GIS GUIDs and compare the normalized facts with fixture expectations. Do not make this a release gate because the public source can be unavailable or change independently.

- [ ] **Step 7: Commit documentation and any verification-only fixes**

```bash
git add README.md gar-init/README.md
git commit -m "docs: describe enriched house data"
```

- [ ] **Step 8: Apply verification-before-completion**

Record the exact successful commands and their exit status. Report any environment limitation separately; never describe an unrun command as passing.

---

## Completion Criteria

- Arbat 4 resolves with the canonical GAR display address including `стр. 1`.
- The detail card exposes observed type, status, years, project, condition, lifecycle, wear, wall, and management facts.
- The square summary supplies residential/non-residential premise counts and areas when available.
- Floors and entrances remain null for the tested house because the public sources do not publish them.
- Optional square failures produce a fresh usable detail profile and an unavailable source marker.
- Existing flat API consumers continue to receive the same keys.
- The My House page displays expanded passport, management, provenance, stale state, and the exact missing-data copy.
- Raw GIS payloads remain server-side and bounded.
- Migration 007 applies after migration 006 and existing cache rows remain readable until refreshed.
- Backend, GAR, frontend, Playwright, build, vet, and Compose checks have recorded outcomes.
