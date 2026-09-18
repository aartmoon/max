# Enriched GIS Housing Profile Design

## Goal

Collect, normalize, cache, and display the maximum useful set of house facts
that is already available from the public GIS Housing detail response, plus the
public house-square summary that supplies premise counts. Keep the profile
usable when optional upstream data is missing or temporarily unavailable.

This is the first stage of a broader house-data aggregator. Resource suppliers,
capital-repair programs, engineering equipment, tariffs, and request routing
remain separate later stages because they use different registries and failure
modes.

## Current Flow and Problems

The existing flow is:

```text
GAR address selection
    -> GAR/FIAS house GUID
    -> GIS Housing lookup by FIAS GUID
    -> GIS Housing detail card
    -> cached gis_house_profiles row
    -> /api/houses/resolve
    -> My House page
```

The integration already retains the successful detail JSON in `raw_payload`,
but normalizes only cadastral number, two areas, floor/entrance/apartment
counts, one year, and limited management contacts.

The live public contract differs from several current DTO assumptions:

- current detail cards use `floorCount`; the adapter expects `floorCountMax`;
- current detail cards expose `residentialPremiseActualCount` and
  `residentialPremiseConfirmedCount`, while the adapter expects
  `residentialPremiseCount`;
- the separate public `get-house-square-data/{gisHouseGuid}` response contains
  authoritative premise counts and squares even when the detail card omits
  them;
- entrance information is not available from the tested anonymous public
  endpoints for the selected Arbat house;
- the GAR fixture for the selected object contains structure information that
  the current search projection drops, producing `д. 4` instead of
  `д. 4, стр. 1`.

Missing upstream values must remain null. The application must not substitute
zeroes, demo values, or inferred facts.

## Selected Approach

Use a staged typed aggregator while retaining raw upstream payloads.

1. Extend the existing GIS Housing adapter to normalize every useful field in
   the lookup/detail JSON that has a stable observed meaning.
2. Fetch the public square summary after a successful detail card and merge its
   premise facts into the same profile.
3. Treat the square summary as optional: a failure must not discard a valid
   detail profile.
4. Persist typed fields for product use and both successful raw payloads for
   diagnosis and future remapping.
5. Return a structured response while retaining the existing flat response
   fields during migration.
6. Render expanded house and management sections in the current page. Clearly
   distinguish unpublished facts from transport failures.

Rejected alternatives:

- Returning raw GIS JSON to the browser would leak an unstable vendor contract
  throughout the product and make future remapping harder.
- Building suppliers, capital repair, equipment, and tariffs in the same change
  would couple independent sources and make a partially useful passport fail
  as one unit.
- Filling missing facts from arbitrary third-party property sites would weaken
  provenance and could show contradictory information.

## Normalized Data Model

### House characteristics

The normalized profile adds nullable typed fields for:

- cadastral number;
- GIS house type code and display name;
- house status;
- total, residential, and non-residential area;
- residential and non-residential premise counts and their summed areas;
- residential premises linked to realty count and area;
- non-residential premises excluding common property count and area;
- floor and entrance count;
- building, operation, and reconstruction years;
- project series;
- building condition;
- lifecycle stage;
- deterioration percentage and its measurement date;
- wall material;
- energy-efficiency class;
- number of owners/shares when published by the square summary.

Existing compatibility fields keep their meanings:

- `apartments` maps to the best published residential-premise count, preferring
  the square summary over detail-card variants;
- `yearBuilt` maps to building year, falling back to operation year only for the
  legacy field;
- `floors` and `entrances` remain null when unpublished.

### Management

The normalized management object contains nullable fields for:

- management method;
- organization GIS GUID;
- short and full name;
- postal address;
- phone and website;
- organization type;
- registry organization GUID;
- INN and OGRN;
- chief full name when the detail response publishes its components;
- management-contract start and end dates when present in the house card.

The legacy `organization`, `manager`, and `contact` fields remain populated from
this object for existing clients.

### Source metadata and raw payloads

The cache stores:

- the raw detail-card payload;
- the raw square-summary payload when successful;
- the last successful profile fetch time;
- whether the response is stale because a refresh failed;
- whether the optional square summary was available during that refresh.

Raw payloads stay server-side and are never returned by the public application
API.

## External Adapter

`FetchHouse` remains the single service-level operation. Internally it performs:

1. lookup by GAR/FIAS house GUID;
2. exact FIAS GUID match and GIS identity extraction;
3. detail-card fetch;
4. detail normalization and merge of lookup fields missing from detail;
5. square-summary fetch using the internal GIS house GUID;
6. merge of square facts when the optional request succeeds.

The lookup and detail calls remain mandatory. A failure in either preserves the
existing not-found/unavailable behavior. The square-summary call is best effort:
timeouts, non-2xx responses, malformed JSON, or size-limit failures leave those
fields null or use detail-card fallbacks without failing the profile.

All numeric DTO fields accept either JSON numbers or numeric strings. Reference
objects use small adapters that tolerate the observed `value`, specialized
display-name properties, and absent values. Unknown JSON fields remain ignored.

Tests use captured, reduced fixtures shaped like real public responses. Live
GIS Housing calls are not part of automated tests.

## Persistence and Migration

A new additive database migration extends `gis_house_profiles`. Existing rows
remain valid and are refreshed naturally after the 24-hour TTL.

The migration adds nullable typed columns for the new characteristic and
management fields, a nullable `square_payload jsonb`, and a boolean indicating
whether the square summary was available. The existing `raw_payload` continues
to contain the detail card.

Repository select, scan, insert, and conflict-update statements cover every new
column. Integration tests verify both null preservation and round-trip storage
of populated profiles.

## Application API

`POST /api/houses/resolve` keeps all current top-level fields for backward
compatibility and adds:

```json
{
  "characteristics": {
    "houseType": "Многоквартирный",
    "status": "APPROVED",
    "projectSeries": "Индивидуальный проект",
    "condition": "Исправный",
    "lifecycleStage": "Эксплуатация",
    "yearBuilt": 1870,
    "operationYear": 1870,
    "reconstructionYear": 1870,
    "deteriorationPercent": 66,
    "wallMaterial": "Стены кирпичные",
    "energyEfficiency": null,
    "totalArea": 6720.8,
    "livingArea": 2405.3,
    "nonResidentialArea": 4023.6,
    "residentialPremises": 15,
    "nonResidentialPremises": 31,
    "floors": null,
    "entrances": null
  },
  "management": {
    "method": "УО",
    "shortName": "ГБУ \"Жилищник района Арбат\"",
    "fullName": "ГОСУДАРСТВЕННОЕ БЮДЖЕТНОЕ УЧРЕЖДЕНИЕ ГОРОДА МОСКВЫ \"ЖИЛИЩНИК РАЙОНА АРБАТ\"",
    "phone": "74952305787",
    "website": "https://arbatgbu.mos.ru/",
    "address": "121099, г. Москва, пер. Проточный, д. 9, стр. 1",
    "inn": null,
    "ogrn": "5147746267906"
  },
  "dataSources": [
    {"name": "ГИС ЖКХ · карточка дома", "available": true},
    {"name": "ГИС ЖКХ · сводка помещений", "available": true}
  ]
}
```

All fact values are nullable. Source entries also include their common profile
update time and stale flag. The exact frontend copy uses
`Не опубликовано в ГИС ЖКХ` for null values.

## Frontend

The My House page keeps address selection and adds two expanded panels.

`Паспорт дома` displays identity, type/status, years, project, condition,
areas, premise counts, floors/entrances, deterioration, energy efficiency, and
wall material. Null facts display `Не опубликовано в ГИС ЖКХ`.

`Кто управляет домом` displays management method and all published organization
details. Links and phone values are rendered safely; absent values use the same
missing-data copy.

The page lists source names and the last successful update time. A stale cache
continues to show the existing non-blocking warning. Failure of the optional
square summary does not produce a full-page error; its source is marked
unavailable and only its facts remain missing.

## GAR Address Correction

The GAR search projection must include `add_type1/add_num1` and
`add_type2/add_num2` when building house display names. The formatter maps only
documented or fixture-verified type codes and preserves the existing
`build_num`/`struc_num` handling. Duplicate textual components are not emitted.

The canonical Arbat fixture test must assert that object `67020241` is rendered
as `г. Москва, ул. Арбат, д. 4, стр. 1`. Resolving an existing local house then
updates its address snapshot through the existing upsert path.

## Failure and Cache Behavior

- Fresh cached profiles make no upstream calls.
- A stale profile attempts a full refresh under the existing advisory lock.
- Mandatory lookup/detail failure with an old cache returns the old profile as
  stale.
- Mandatory lookup/detail failure without a cache returns the existing 404/503
  errors.
- Optional square-summary failure still saves and returns the fresh detail
  profile.
- Partial upstream fields remain null and never erase unrelated successful
  fields through application-level defaulting.
- Raw bodies remain bounded by the existing 2 MiB per-response limit.

## Testing

Backend tests cover:

- the current `floorCount` field and legacy `floorCountMax` fallback;
- string and numeric forms for years, counts, areas, and deterioration;
- every added detail-card field using a reduced real-response fixture;
- square-summary precedence for residential premise counts;
- optional square-summary timeout, non-2xx, malformed, and oversized responses;
- persistence round trips for full and partial profiles;
- service cache and stale fallback behavior;
- controller compatibility fields and new structured objects.

GAR initializer tests cover the structure suffix and prevent duplicate suffixes.

Frontend browser tests cover populated and partial profiles, source labels,
safe management links, the new missing-data copy, and continued operation when
the square source is unavailable.

Final verification includes backend and GAR Go tests, `go vet`, frontend build,
targeted Playwright tests, and Compose configuration validation. A read-only
live probe may be used for manual comparison but is not a release gate.

## Explicit Non-Goals for This Stage

- resource-supplier discovery and request routing;
- capital-repair programs;
- lift, porch, meter, or engineering-equipment registries;
- tariffs;
- protected GIS Housing/SOAP integration or credentials;
- exposing raw upstream JSON to browsers;
- fabricating missing facts or using unverified third-party property data.
