CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS gar_import_state (
    file_name text PRIMARY KEY,
    entity_type text NOT NULL,
    file_size bigint NOT NULL CHECK (file_size >= 0),
    row_count bigint NOT NULL CHECK (row_count >= 0),
    started_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_import_metadata (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    initialized boolean NOT NULL DEFAULT false,
    initialized_at timestamptz,
    source_date date,
    source_type text CHECK (source_type IN ('xml', 'demo'))
);

CREATE TABLE IF NOT EXISTS gar_address_objects (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    name text, type_name text, level integer, operation_type_id integer,
    previous_id bigint, next_id bigint, update_date date, start_date date,
    end_date date, is_actual boolean, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_addr_object_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_addr_object_divisions (
    id bigint, parent_id bigint, child_id bigint, change_id bigint,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_adm_hierarchy (
    id bigint, object_id bigint, parent_object_id bigint, change_id bigint,
    region_code text, area_code text, city_code text, place_code text,
    plan_code text, street_code text, path text, update_date date,
    start_date date, end_date date, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_mun_hierarchy (
    id bigint, object_id bigint, parent_object_id bigint, change_id bigint,
    region_code text, area_code text, city_code text, place_code text,
    plan_code text, street_code text, path text, update_date date,
    start_date date, end_date date, is_active boolean, oktmo text,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_houses (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    house_num text, build_num text, struc_num text, house_type integer,
    add_type1 integer, add_num1 text, add_type2 integer, add_num2 text,
    operation_type_id integer, update_date date, start_date date, end_date date,
    is_actual boolean, is_active boolean, raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_house_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_apartments (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    number text, apart_number text, apart_type integer, operation_type_id integer,
    update_date date, start_date date, end_date date, is_actual boolean,
    is_active boolean, raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_apartment_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_carplaces (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    number text, operation_type_id integer, update_date date, start_date date,
    end_date date, is_actual boolean, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_carplace_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_rooms (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    number text, room_type integer, operation_type_id integer, update_date date,
    start_date date, end_date date, is_actual boolean, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_room_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_steads (
    id bigint, object_id bigint, object_guid uuid, change_id bigint,
    number text, operation_type_id integer, update_date date, start_date date,
    end_date date, is_actual boolean, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_stead_params (
    id bigint, object_id bigint, change_id bigint, change_id_end bigint,
    type_id integer, value text, update_date date, start_date date, end_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_change_history (
    change_id bigint, object_id bigint, address_object_id uuid,
    operation_type_id integer, normative_doc_id bigint, change_date date,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_normative_docs (
    id bigint, name text, number text, document_date date, document_type integer,
    document_kind integer, update_date date, organization_name text,
    registration_number text, registration_date date, acceptance_date date,
    comment text, raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_reestr_objects (
    object_id bigint, object_guid uuid, change_id bigint, level_id integer,
    create_date date, update_date date, is_active boolean,
    raw_attributes jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_search_addresses (
    object_id bigint PRIMARY KEY,
    object_guid uuid,
    parent_object_id bigint,
    object_kind text NOT NULL,
    display_name text NOT NULL,
    full_address text NOT NULL,
    search_text text NOT NULL,
    level integer,
    is_active boolean NOT NULL
);

-- Upgrade empty tables created by the retired backend ZIP importer. These
-- ALTER statements are idempotent and preserve any existing data.
-- OBJECTID/OBJECTGUID identify the logical GAR object and repeat across its
-- historical versions. The retired importer incorrectly made them unique.
ALTER TABLE gar_address_objects DROP CONSTRAINT IF EXISTS gar_address_objects_pkey;
ALTER TABLE gar_address_objects DROP CONSTRAINT IF EXISTS gar_address_objects_object_guid_key;
ALTER TABLE gar_adm_hierarchy DROP CONSTRAINT IF EXISTS gar_adm_hierarchy_pkey;
ALTER TABLE gar_houses DROP CONSTRAINT IF EXISTS gar_houses_pkey;
ALTER TABLE gar_houses DROP CONSTRAINT IF EXISTS gar_houses_object_guid_key;

DROP INDEX IF EXISTS gar_address_objects_object_id_uq;
DROP INDEX IF EXISTS gar_houses_object_id_uq;
DROP INDEX IF EXISTS gar_apartments_object_id_uq;
DROP INDEX IF EXISTS gar_carplaces_object_id_uq;
DROP INDEX IF EXISTS gar_rooms_object_id_uq;
DROP INDEX IF EXISTS gar_steads_object_id_uq;

-- ADROBJECTID in GAR change history is a GUID. Early versions of this
-- importer declared it as bigint, which rejects current GAR snapshots.
ALTER TABLE gar_change_history
    ALTER COLUMN address_object_id TYPE uuid
    USING address_object_id::text::uuid;

ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS id bigint;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS change_id bigint;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS operation_type_id integer;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS previous_id bigint;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS next_id bigint;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS update_date date;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS start_date date;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS end_date date;
ALTER TABLE gar_address_objects ADD COLUMN IF NOT EXISTS raw_attributes jsonb NOT NULL DEFAULT '{}';

ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS id bigint;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS change_id bigint;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS region_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS area_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS city_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS place_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS plan_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS street_code text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS path text;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS update_date date;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS start_date date;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS end_date date;
ALTER TABLE gar_adm_hierarchy ADD COLUMN IF NOT EXISTS raw_attributes jsonb NOT NULL DEFAULT '{}';

ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS id bigint;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS change_id bigint;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS build_num text;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS struc_num text;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS operation_type_id integer;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS update_date date;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS start_date date;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS end_date date;
ALTER TABLE gar_houses ADD COLUMN IF NOT EXISTS raw_attributes jsonb NOT NULL DEFAULT '{}';
ALTER TABLE gar_houses ALTER COLUMN house_num DROP NOT NULL;
ALTER TABLE gar_houses ALTER COLUMN add_num1 DROP NOT NULL;
ALTER TABLE gar_houses ALTER COLUMN add_num2 DROP NOT NULL;
