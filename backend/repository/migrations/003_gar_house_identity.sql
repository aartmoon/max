CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE houses ADD COLUMN IF NOT EXISTS fias_guid uuid;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS cadastral_number text;
ALTER TABLE houses ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE houses ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS ux_houses_fias_guid ON houses(fias_guid) WHERE fias_guid IS NOT NULL;

CREATE TABLE IF NOT EXISTS gar_address_objects (
 object_id bigint PRIMARY KEY,
 object_guid uuid NOT NULL UNIQUE,
 name text NOT NULL,
 type_name text NOT NULL,
 level integer NOT NULL,
 is_active boolean NOT NULL,
 is_actual boolean NOT NULL,
 gar_updated_at date
);

CREATE TABLE IF NOT EXISTS gar_adm_hierarchy (
 object_id bigint PRIMARY KEY,
 parent_object_id bigint,
 is_active boolean NOT NULL,
 gar_updated_at date
);

CREATE TABLE IF NOT EXISTS gar_house_types (
 type_id integer PRIMARY KEY,
 name text NOT NULL,
 short_name text NOT NULL,
 is_active boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_param_types (
 type_id integer PRIMARY KEY,
 name text NOT NULL,
 code text NOT NULL,
 is_active boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS gar_params (
 object_id bigint NOT NULL,
 type_id integer NOT NULL,
 value text NOT NULL,
 gar_updated_at date,
 PRIMARY KEY(object_id, type_id)
);

CREATE TABLE IF NOT EXISTS gar_houses (
 object_id bigint PRIMARY KEY,
 object_guid uuid NOT NULL UNIQUE,
 house_num text NOT NULL,
 add_num1 text NOT NULL,
 add_num2 text NOT NULL,
 house_type integer,
 add_type1 integer,
 add_type2 integer,
 is_active boolean NOT NULL,
 is_actual boolean NOT NULL,
 gar_updated_at date
);

CREATE TABLE IF NOT EXISTS addresses (
 fias_guid uuid PRIMARY KEY,
 gar_object_id bigint NOT NULL UNIQUE,
 address text NOT NULL,
 search_text text NOT NULL,
 object_type text NOT NULL,
 region text NOT NULL DEFAULT '',
 area text NOT NULL DEFAULT '',
 city text NOT NULL DEFAULT '',
 settlement text NOT NULL DEFAULT '',
 street text NOT NULL DEFAULT '',
 house text NOT NULL DEFAULT '',
 postal_code text,
 cadastral_number text,
 is_active boolean NOT NULL,
 gar_updated_at date,
 imported_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS addresses_search_trgm_idx ON addresses USING gin(search_text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS addresses_active_type_idx ON addresses(object_type, is_active);
