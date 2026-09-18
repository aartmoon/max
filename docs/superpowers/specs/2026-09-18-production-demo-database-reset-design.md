# Production Demo Database Reset Design

## Goal

Replace the full production GAR/FIAS dataset and all current application data
with a small, reproducible demo database centered on ten real houses on Moscow's
Arbat street. The same fixture must initialize local and production databases,
must support the application's address APIs, and must contain real GAR object
identifiers and GUIDs suitable for calls to the GIS Housing registry.

The reset must avoid transformations over the full Russian address dataset.
The destructive production step is a constant-time schema reset followed by
creation and seeding of a small database.

## Scope

The reset deletes all data in the production database, including requests,
request photos, users, resolved houses, cached GIS Housing profiles, migration
state, GAR import state, and the full GAR/FIAS snapshot.

The replacement database contains:

- the Moscow and Arbat address-object ancestry required to render full
  addresses;
- ten active, actual houses directly associated with Arbat;
- a deterministic sample of available apartments, rooms, car places, and
  steads associated with those houses or the retained street;
- every historical version and parameter row for each selected object;
- retained administrative and municipal hierarchy rows needed to connect the
  selected objects to their parents;
- address-object division rows whose two endpoints are retained;
- registry and change-history rows for retained objects;
- normative documents referenced by retained change-history rows;
- a search index rebuilt from the retained source rows;
- the application's existing demo user and organizations.

The fixture does not preserve production requests or other user-generated
application data.

## Chosen Approach

Use the existing `gar-init` service as the single owner of the GAR schema and
demo fixture. Add an explicit demo mode that never scans the mounted full GAR
XML directory. Both local and production Compose configurations run this mode
before the backend and wait for successful completion.

This is preferred over the alternatives:

1. Filtering the full database in place would require large deletes, index
   maintenance, vacuuming, and long-running transactions on the constrained VM.
2. Seeding only `gar_search_addresses` would make address search work but would
   not retain representative data across all GAR tables and could drift from
   the normal search-building logic.
3. Keeping independent local and production seeds would allow the two
   environments to diverge.

The fixture is a versioned artifact in the repository. It is extracted once
from production before the reset and is then independent of the production
database.

## Fixture Selection

### Street identity

The extractor must identify exactly one active, actual level-8 address object
named `Арбат` whose complete administrative address is Moscow. Selection must
not rely only on the street name because GAR contains namesakes in other
regions.

### Houses

Select ten active, actual houses whose active administrative parent is the
chosen Arbat object. Candidate ordering is deterministic. Prefer houses that:

- have non-null GAR GUIDs;
- are returned successfully by the existing GIS Housing client;
- collectively have examples of apartments, rooms, and car places in GAR;
- have useful parameter, registry, or change-history rows.

At least two selected houses must successfully resolve through the real GIS
Housing integration before the fixture is accepted. External GIS responses are
not copied into the fixture; the verification proves that the retained real
GUIDs can drive the live integration.

### Descendant samples

For each selected house, retain deterministic samples where data exists:

- up to five active, actual apartments;
- up to two active, actual rooms;
- up to two active, actual car places.

Retain up to two active, actual steads associated with the selected street when
available. An entity family may be empty only when the source database has no
matching entity for any selected house; the extractor must report this
explicitly rather than fabricate data.

For every selected logical object ID, retain all matching rows from its entity
table and all matching rows from its parameter table. This preserves historical
versions and all available attributes while keeping the number of logical
objects bounded.

### Relationship closure

Build a retained-object set containing the Moscow ancestry, Arbat, selected
houses, and sampled descendants. Copy:

- all administrative and municipal hierarchy rows for retained objects and
  required ancestors;
- address-object divisions only when both endpoints are retained;
- all reestr-object rows for retained object IDs;
- all change-history rows for retained object IDs;
- every normative document referenced by retained change-history rows.

Do not copy `gar_import_state` from the full snapshot. Create fresh metadata
that labels the database as the Arbat demo fixture and records the source GAR
date and fixture generation time.

## Fixture Format and Validation

Store the fixture in a reviewable, deterministic SQL file owned by `gar-init`.
It contains only literal inserts into GAR source tables; schema creation,
indexes, and search-index construction continue to use the existing schema and
finalization code.

The extraction command writes to a temporary file first, validates it, and only
then replaces the tracked fixture. Validation must prove:

- exactly one retained Arbat street and exactly ten retained houses;
- all retained GUIDs parse as UUIDs and all logical object IDs are positive;
- every sampled child has a retained hierarchy path to a selected house or the
  retained street;
- all parameter, registry, history, division, and normative-document rows
  reference retained objects or documents as required;
- search finalization succeeds on an empty database;
- address search returns Arbat houses and descendant entities with full Moscow
  addresses;
- the fixture is byte-for-byte deterministic when generated twice from the
  same source.

## Runtime Initialization

Introduce an explicit GAR operating mode:

- `demo`: create the GAR schema, load the embedded Arbat fixture when the GAR
  database is empty, build indexes and `gar_search_addresses`, and exit;
- `full`: retain the existing XML import behavior for an explicit future need,
  but never select it implicitly in local or production Compose.

Both `docker-compose.yml` and `docker-compose.prod.yml` set demo mode. The
backend waits for `gar-init` to complete successfully. A repeated start is
idempotent: initialized demo metadata causes `gar-init` to exit without
inserting duplicates or rebuilding the search index.

No startup path reads the production XML mount in demo mode. This prevents a
fresh server or empty volume from accidentally importing all Russian addresses
again.

## Production Reset Procedure

The reset is an explicit operator action, not an automatic migration.

1. Build and test the new backend and `gar-init` images.
2. Stop the production backend and `gar-init` services so no process holds the
   migration or GAR advisory lock.
3. Extract the Arbat fixture from the still-intact production database to a
   file outside the PostgreSQL volume.
4. Copy the fixture off the VM, validate it in a fresh local database, and
   commit it to the repository.
5. Create a full compressed database dump or provider volume snapshot when
   storage permits. The validated fixture is mandatory; the full backup is the
   rollback artifact for non-address production data.
6. Deploy the image containing the validated fixture while keeping application
   traffic stopped.
7. Reset PostgreSQL with `DROP SCHEMA public CASCADE`, recreate `public` owned
   by the application database owner, and restore the normal schema privileges.
8. Start `gar-init` in demo mode and wait for exit code zero.
9. Start the backend and frontend.
10. Run the verification checks below before restoring traffic.

The reset command must require the operator to supply the expected database
name explicitly and must abort when it does not match. It must print the target
database and fixture checksum before asking for a typed confirmation. It must
not run as part of application startup or a normal deploy.

## Verification

Automated tests cover demo-mode selection, refusal to mix demo and full modes,
idempotent seeding, per-table fixture counts, relationship closure, search
construction, and backend address lookup/house-resolution behavior.

Production smoke checks verify:

- `/api/health` returns success;
- searching for `Арбат` returns only the retained demo scope;
- filtering by parent returns houses and sampled descendants correctly;
- `GET /api/addresses/{objectId}` returns the retained real GUID;
- `POST /api/houses/resolve` succeeds for at least two pre-verified houses and
  returns data sourced from GIS Housing;
- a demo request can be created for a retained house and, where present, a
  retained apartment;
- no address outside the fixture is returned;
- all tables are small and the full Russian GAR snapshot is absent.

## Failure and Recovery

No production data is deleted until the extracted fixture passes validation in
a fresh local database and exists outside the VM.

If fixture generation or validation fails, production remains unchanged. If
initialization fails after the schema reset, keep traffic stopped, inspect the
small-database error, and retry initialization. If restoring the previous
production state is required, recreate the database from the full dump or
volume snapshot. The fixture alone restores only the intended demo state, not
deleted user data.

## Non-Goals

- Keeping the full GAR XML snapshot online.
- Supporting arbitrary Russian address search in demo mode.
- Periodically synchronizing the demo fixture from GAR.
- Copying live GIS Housing responses into source control.
- Preserving existing production requests, photos, or cached profiles.
