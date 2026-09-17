CREATE TABLE IF NOT EXISTS gis_house_profiles (
    house_id bigint PRIMARY KEY REFERENCES houses(id) ON DELETE CASCADE,
    gis_house_guid uuid NOT NULL,
    gis_house_type text NOT NULL,
    cadastral_number text,
    total_area double precision,
    living_area double precision,
    floors integer,
    entrances integer,
    apartments integer,
    year_built integer,
    organization text,
    manager text,
    contact text,
    raw_payload jsonb NOT NULL,
    fetched_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
