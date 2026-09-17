# GIS Housing Profile Design

## Goal

Replace the fixed demo passport on the `MyHouse` page with a resident-selected
house and published housing data from the public part of GIS Housing and
Utilities (GIS ЖКХ).

The application already imports GAR/FIAS data into PostgreSQL and lets a user
select a house by GAR `objectId`. The new flow reuses that selection and the
house `objectGuid` to find the corresponding public GIS ЖКХ record, fetch its
published profile, normalize the fields needed by the product, and cache the
result locally.

This integration does not use the protected SOAP exchange. It consumes the
same unauthenticated JSON endpoints used by the public GIS ЖКХ housing registry.
Those endpoints are not a documented external API, so the integration treats
their contract as unstable and isolates it behind a dedicated adapter.

## User Experience

`MyHouse` has two states:

1. If no house is selected in the current browser, show the existing GAR house
   autocomplete and ask the user to choose a house.
2. After selection, show the house address immediately and load the housing
   profile through the application backend.

The browser stores only the selected GAR `objectId` in `localStorage`. This is
necessary while authentication and resident identity are absent: persisting a
selection against the shared demo user would allow one visitor to change the
house for every other visitor.

The page allows the user to change or clear the selected house. Clearing it
removes the local selection but does not delete the shared cached profile.

Missing published fields are rendered as `Нет данных`. The page no longer
labels the selected house as a test house and does not substitute demo values.
It displays the profile source and the last successful update time.

## Application API

The frontend reuses `POST /api/houses/resolve` with the existing request body:

```json
{"objectId":"123456"}
```

The endpoint continues to validate the GAR identifier and resolve the local
`houses` row. It then loads the cached GIS ЖКХ profile or refreshes it when
needed. The successful response contains the selected GAR identity, normalized
profile fields, and cache metadata:

```json
{
  "id": "42",
  "garObjectId": "123456",
  "objectGuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
  "address": "г. Москва, ул. Примерная, д. 1",
  "cadastralNumber": "77:00:0000000:1",
  "totalArea": 10000.5,
  "livingArea": 7500.1,
  "floors": 12,
  "entrances": 4,
  "apartments": 180,
  "yearBuilt": 1987,
  "organization": "Управляющая организация",
  "manager": "",
  "contact": "+7 ...",
  "dataSource": "gis-housing",
  "dataUpdatedAt": "2026-09-18T09:00:00Z",
  "stale": false
}
```

Fields absent from GIS ЖКХ are returned as `null`; the frontend `House` type is
updated accordingly and renders them as missing data. This avoids confusing a
missing year, count, or area with a real zero. `dataUpdatedAt` is the time of the
last successful refresh.

`GET /api/house`, which currently returns the fixed address
`г. Москва, ул. Тестовая, д. 1`, is removed along with its fixed-address
repository query. No endpoint may inject demo house characteristics into the
new flow.

## External GIS ЖКХ Adapter

A dedicated backend interface owns all access to the public GIS ЖКХ registry.
Its implementation uses a fixed HTTPS origin and does not accept a URL from a
request.

The observed public flow is:

1. Use the GAR/FIAS house GUID with the public
   `houses/searchByFiasHouseCodeList` JSON endpoint.
2. Select the exact published house match and obtain the internal GIS ЖКХ house
   GUID and house type code.
3. Fetch the public house card by house type code and internal house GUID.
4. Map only product fields into an internal DTO and retain the raw payload for
   diagnosis and future remapping.

The adapter uses explicit connect and response timeouts, bounded response
bodies, and a product-specific User-Agent. It accepts unknown response fields
but fails clearly when required identity fields are absent or the response is
not valid JSON. It does not parse the public HTML pages.

Because the endpoints are an inferred internal contract rather than a supported
external API, endpoint paths and response DTOs remain confined to the adapter.
Controllers, services, repositories, and frontend code do not depend on the
upstream response shape.

## Persistence and Caching

Create a `gis_house_profiles` table keyed by application `house_id`. It stores:

- GIS ЖКХ internal house GUID and house type code;
- normalized cadastral number, areas, floor/entrance/apartment counts, year,
  management organization, manager, and contact fields;
- the raw upstream payload as `jsonb`;
- `fetched_at`, `created_at`, and `updated_at` timestamps.

The existing `houses` table remains the source of GAR identity and canonical
address. GIS ЖКХ management organizations are not inserted into the existing
`organizations` table because that table currently controls demo request
routing; mixing the two would change routing behavior unintentionally.

A profile is fresh for 24 hours. Resolving a house behaves as follows:

1. Validate and resolve the current GAR house as today.
2. Return a cached profile immediately when it is fresh.
3. When missing or stale, request the public GIS ЖКХ data and upsert the profile
   transactionally.
4. Return the refreshed profile.

Concurrent refreshes for one house are serialized with a PostgreSQL advisory
lock so opening the same house across multiple backend instances does not fan
out identical upstream requests. The cache is shared across visitors; the
browser selection is not.

## Failure Handling

If refresh fails and a cached profile exists, return that profile with
`stale: true` and its original `dataUpdatedAt`. The UI displays a non-blocking
warning that current data could not be refreshed.

If no cached profile exists:

- a valid GAR house with no GIS ЖКХ match returns `404` with a user-facing
  message that published house information was not found;
- upstream timeout, transport failure, invalid response, or temporary server
  failure returns `503` with a retryable user-facing message;
- invalid or inactive GAR identity continues to use the existing validation
  errors.

Logs include the local house ID, GAR GUID, operation, status, and duration, but
not full upstream bodies. The raw successful payload is available in the cache
for diagnosis. `MOCK_STATUS_ENABLED` remains unchanged and continues to control
only the resident demo status-transition endpoint and UI.

## Components

Backend additions:

- `GISRegistryClient` interface and HTTP implementation in `backend/integration`;
- `HouseProfileService` coordinating validation, cache, refresh, and fallback;
- repository methods and a migration for `gis_house_profiles`;
- the extended `/api/houses/resolve` response.

Frontend changes:

- reuse `AddressAutocomplete` on `MyHouse` for `kind="house"`;
- store and restore the selected GAR `objectId` in `localStorage`;
- call the extended resolve endpoint and render loading, missing-data, stale,
  empty-selection, and retry states;
- remove test-house labels and the demo passport note from this page.

Configuration adds an overridable GIS ЖКХ base URL for tests and deployments,
defaulting to the fixed official HTTPS origin, plus request timeout and cache
TTL values with conservative defaults. No credentials are introduced.

## Testing

Unit tests use an in-process fake HTTP server or injected fake client; automated
tests never depend on the live GIS ЖКХ site.

Backend coverage includes:

- exact GAR GUID to GIS ЖКХ identity matching;
- normalized mapping for complete and partial house cards;
- malformed, oversized, timeout, not-found, and upstream-error responses;
- fresh-cache avoidance of external calls;
- stale-cache refresh and stale fallback;
- first-load failure without fabricated demo data;
- repository upsert and migration behavior.

Frontend browser tests include:

- first visit prompts for a house;
- selection persists only in the current browser storage;
- reload resolves and renders the saved house;
- changing and clearing the selection;
- partial data renders `Нет данных`;
- stale and retryable error states;
- no request to the legacy `/api/house` endpoint.

Final verification runs all backend tests and vetting, frontend build and browser
tests, Compose configuration validation, and a manual smoke request against a
controlled fake upstream. A live GIS ЖКХ probe is an optional deployment check,
not a release-blocking automated test.

## Explicit Non-Goals

- SOAP integration, organization registration, certificates, or signing;
- parsing GIS ЖКХ HTML pages;
- resident authentication or server-side user-to-house ownership;
- using GIS ЖКХ organizations for request routing;
- background crawling or bulk mirroring of the GIS ЖКХ registry;
- guaranteeing availability of undocumented upstream endpoints.
