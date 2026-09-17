# GAR Init Design

## Goal

Build a standalone Go 1.25+ Docker job that imports the Moscow GAR/FIAS XML
snapshot from `/data/gar/*.XML` into PostgreSQL once, exits successfully, and
provides normalized address data for house and apartment autocomplete in the
backend.

Backward compatibility with the existing ZIP importer and its database schema
is not required. The old importer will be removed or replaced.

## Inputs

The job recognizes exactly these filename families, case-insensitively:

- `AS_ADDR_OBJ_*.XML`
- `AS_ADDR_OBJ_PARAMS_*.XML`
- `AS_ADDR_OBJ_DIVISION_*.XML`
- `AS_ADM_HIERARCHY_*.XML`
- `AS_MUN_HIERARCHY_*.XML`
- `AS_HOUSES_*.XML`
- `AS_HOUSES_PARAMS_*.XML`
- `AS_APARTMENTS_*.XML`
- `AS_APARTMENTS_PARAMS_*.XML`
- `AS_CARPLACES_*.XML`
- `AS_CARPLACES_PARAMS_*.XML`
- `AS_ROOMS_*.XML`
- `AS_ROOMS_PARAMS_*.XML`
- `AS_STEADS_*.XML`
- `AS_STEADS_PARAMS_*.XML`
- `AS_CHANGE_HISTORY_*.XML`
- `AS_NORMATIVE_DOCS_*.XML`
- `AS_REESTR_OBJECTS_*.XML`

Files are ordinary uncompressed XML files mounted read-only. Unknown files are
ignored and logged. A run fails if the directory contains no supported files or
if any required family from the manifest above is absent.

## Runtime Configuration

Required environment variables:

- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`

Optional variables:

- `GAR_DATA_DIR`, default `/data/gar`
- `GAR_BATCH_SIZE`, default `10000`
- `GAR_LOG_EVERY`, default `100000`

The PostgreSQL connection uses `sslmode=disable` inside the private Compose
network. Configuration validation happens before connecting or reading files.

## Components

The standalone `gar-init` module has these boundaries:

- `cmd/gar-init`: process lifecycle, signals, logging, and exit codes.
- `internal/config`: environment parsing and validation.
- `internal/model`: XML family declarations, table mappings, typed row values,
  and column definitions.
- `internal/importer`: deterministic file discovery, streaming XML decoding,
  attribute conversion, bounded batches, progress reporting, and orchestration.
- `internal/db`: schema creation, advisory locking, transactions, `CopyFrom`,
  import state, post-import indexes, and search-index construction.

The existing backend no longer owns the raw GAR import. It only queries the
finished search tables and persists the selected GAR identifiers with the
application's house or resident data.

## Streaming and Memory Bound

Each XML file is opened directly and processed with `encoding/xml.Decoder.Token`.
Only start-element attributes are inspected. Child element names such as
`OBJECT`, `HOUSE`, `APARTMENT`, `ROOM`, `CARPLACE`, `STEAD`, `PARAM`, and `ITEM`
are accepted according to their filename family; the parser does not unmarshal
the document.

Rows are accumulated in a reusable slice capped by `GAR_BATCH_SIZE`. Every full
slice is written with `pgx.Tx.CopyFrom`, then reset while retaining its bounded
capacity. With the default batch size, memory is proportional to 10,000 rows,
not to a 1-3 GB source file. All non-specialized XML attributes are retained in
`raw_attributes jsonb` so later schema refinements do not require reacquiring the
snapshot.

## Database Schema

The importer creates one raw table for each supported XML family:

- `gar_address_objects`
- `gar_addr_object_params`
- `gar_addr_object_divisions`
- `gar_adm_hierarchy`
- `gar_mun_hierarchy`
- `gar_houses`
- `gar_house_params`
- `gar_apartments`
- `gar_apartment_params`
- `gar_carplaces`
- `gar_carplace_params`
- `gar_rooms`
- `gar_room_params`
- `gar_steads`
- `gar_stead_params`
- `gar_change_history`
- `gar_normative_docs`
- `gar_reestr_objects`

Common GAR identifiers are typed as follows:

- `OBJECTID`, `PARENTOBJID`, `CHANGEID`, and row `ID`: `bigint`
- `OBJECTGUID`: `uuid`
- activity and actuality flags: `boolean`
- GAR dates: `date`
- names, numbers, paths, codes, and parameter values: `text`
- attributes without first-class columns: `jsonb`

Typed columns include every applicable occurrence of `OBJECTID`, `OBJECTGUID`,
`PARENTOBJID`, `NAME`, `TYPENAME`, `LEVEL`, `HOUSENUM`, `BUILDNUM`, `STRUCNUM`,
`APARTNUMBER`, `NUMBER`, `ISACTUAL`, and `ISACTIVE`.

`gar_import_state` stores the filename, entity family, byte size, imported row
count, and timestamps. `gar_import_metadata` stores the singleton completion
marker and source snapshot date. `gar_search_addresses` is a derived,
denormalized autocomplete table containing object ID, GUID, parent ID, kind,
display label, full address, normalized search text, level, and active status.

The canonical DDL lives in `gar-init/internal/db/schema.sql`. Raw tables are
created without secondary indexes before the bulk load. Idempotent unique and
search indexes are added only after every file transaction has committed.

No runtime path uses `DROP TABLE`, `TRUNCATE`, or deletion of successfully
imported data.

## Import Ordering

File discovery sorts by a fixed dependency order rather than filesystem order:

1. address objects and their parameters/divisions;
2. administrative and municipal hierarchy;
3. houses, apartments, car places, rooms, and steads;
4. each object's parameter files;
5. change history, normative documents, and reestr objects.

The search table is populated only after all raw files have committed. Address
chains use the active administrative hierarchy. Both administrative and
municipal hierarchy remain available in raw form.

## Transactions, Restart, and Concurrency

The process checks out one dedicated `pgxpool.Conn` and acquires a PostgreSQL
session advisory lock before inspecting import state. This prevents two jobs
from importing concurrently. The lock is released on normal exit and by
PostgreSQL if the process or connection dies.

Every XML file is handled in one database transaction:

1. check `gar_import_state`;
2. stream and `COPY` all batches into the destination table;
3. insert the completed state row;
4. commit.

If parsing or copying fails, the entire file transaction rolls back and no state
row is recorded. On restart, completed files are skipped and the failed file
starts from byte zero. XML decoder byte offsets are deliberately not persisted,
because PostgreSQL and XML cannot atomically checkpoint a partial file.

After all files are complete, the job creates indexes, rebuilds the derived
search table transactionally, records `initialized=true`, and exits with code 0.
If index or search-table construction fails, all raw-file state remains valid;
the next run retries only finalization. Once initialized, a later run logs
`GAR database already initialized` and exits with code 0 without scanning XML.

The state identity is filename plus file size. A changed file with the same name
causes a hard error rather than silently mixing snapshots. Applying future GAR
deltas or replacing an initialized snapshot is outside this one-time importer.

## Address Search and Backend Integration

`gar_search_addresses` contains active address objects, houses, apartments,
rooms, car places, and steads. Parent-child relations come from
`gar_adm_hierarchy`. Search labels use entity-specific fields (`NAME`,
`HOUSENUM`, `BUILDNUM`, `STRUCNUM`, `APARTNUMBER`, or `NUMBER`). Full addresses
are derived from the active hierarchy path.

The backend autocomplete API returns at least:

- `objectId`
- `objectGuid`
- `parentObjectId`
- `objectKind`
- `displayName`
- `fullAddress`

Queries filter active rows and rank prefix matches before trigram matches. A
parent ID may be supplied to support stepwise navigation from street to house
and from house to apartment. Selected houses and apartments are stored by GAR
object ID and GUID rather than only by display text.

## Compose Lifecycle

`gar-init` is an ephemeral Compose service with `restart: "no"`. It depends on
the healthy PostgreSQL service, mounts the host XML directory at `/data/gar:ro`,
and stores no local state. The backend depends on `gar-init` with
`condition: service_completed_successfully`, because this deployment requires
autocomplete data before the backend becomes available.

The production Compose file exposes the host path as a configurable variable,
defaulting to `/home/user1/gar/xml`.

## Logging and Failures

Logs use the required form:

```text
[GAR] importing AS_HOUSES_20260910_....XML
[GAR] AS_HOUSES: 100000 rows
[GAR] finished AS_HOUSES: 853421 rows in 18.2s
```

Errors include the filename, XML decoder position when available, destination
table, and operation. A configuration, connection, schema, lock, discovery,
parse, copy, commit, indexing, or finalization error produces a non-zero exit
code.

## Testing and Verification

Unit tests cover configuration, filename precedence (especially
`AS_ADDR_OBJ_PARAMS` before `AS_ADDR_OBJ` and `AS_HOUSES_PARAMS` before
`AS_HOUSES`), typed attribute parsing, alternate child element names, malformed
XML, batching, cancellation, and progress counts.

Integration tests use PostgreSQL to verify `CopyFrom`, per-file rollback,
restart skipping, state mismatch detection, advisory locking, index creation,
and search-address generation. Compose configuration is rendered and validated.
The final verification runs all new Go tests, existing backend tests, and Docker
build/config checks.
