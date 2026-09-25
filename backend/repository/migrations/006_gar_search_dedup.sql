DO $$
BEGIN
IF to_regclass('gar_search_addresses') IS NOT NULL THEN
    WITH ranked AS (
        SELECT ctid,
               ROW_NUMBER() OVER (
                   PARTITION BY object_kind, lower(full_address)
                   ORDER BY object_guid NULLS LAST, object_id
               ) AS duplicate_rank
        FROM gar_search_addresses
    )
    DELETE FROM gar_search_addresses current_row
    USING ranked
    WHERE current_row.ctid = ranked.ctid
      AND ranked.duplicate_rank > 1;
END IF;
END $$;

DO $$
BEGIN
IF to_regclass('gar_search_addresses') IS NOT NULL THEN
    CREATE UNIQUE INDEX IF NOT EXISTS gar_search_addresses_kind_address_uq
        ON gar_search_addresses (object_kind, lower(full_address));
END IF;
END $$;
