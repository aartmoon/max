DELETE FROM gar_search_addresses;

WITH RECURSIVE active_hierarchy AS (
    SELECT DISTINCT ON (object_id) object_id, parent_object_id
    FROM gar_adm_hierarchy
    WHERE is_active IS TRUE
    ORDER BY object_id, update_date DESC NULLS LAST, id DESC NULLS LAST
),
node_candidates AS (
    SELECT object_id, object_guid, 'address_object'::text AS object_kind,
           trim(concat_ws(' ', type_name, name)) AS display_name,
           level, true AS is_active
    FROM gar_address_objects
    WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
    UNION ALL
    SELECT object_id, object_guid, 'house',
           concat_ws(', ', trim(concat_ws(' ', 'д.', house_num)),
                CASE WHEN build_num IS NOT NULL AND build_num <> '' THEN 'корп. ' || build_num END,
                CASE WHEN struc_num IS NOT NULL AND struc_num <> '' THEN 'стр. ' || struc_num END,
                CASE WHEN add_type1 = 2 AND nullif(add_num1, '') IS NOT NULL
                           AND add_num1 IS DISTINCT FROM struc_num
                     THEN 'стр. ' || add_num1 END,
                CASE WHEN add_type2 = 2 AND nullif(add_num2, '') IS NOT NULL
                           AND add_num2 IS DISTINCT FROM struc_num
                           AND add_num2 IS DISTINCT FROM add_num1
                     THEN 'стр. ' || add_num2 END),
           10, true
    FROM gar_houses
    WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
    UNION ALL
    SELECT object_id, object_guid, 'apartment',
           trim(concat_ws(' ', 'кв.', coalesce(nullif(apart_number, ''), number))),
           11, true
    FROM gar_apartments
    WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
    UNION ALL
    SELECT object_id, object_guid, 'room', trim(concat_ws(' ', 'комн.', number)), 12, true
    FROM gar_rooms WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
    UNION ALL
    SELECT object_id, object_guid, 'carplace', trim(concat_ws(' ', 'м/м.', number)), 13, true
    FROM gar_carplaces WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
    UNION ALL
    SELECT object_id, object_guid, 'stead', trim(concat_ws(' ', 'уч.', number)), 9, true
    FROM gar_steads WHERE is_active IS TRUE AND is_actual IS TRUE AND object_id IS NOT NULL
),
nodes AS (
    SELECT DISTINCT ON (object_kind, object_id) object_id, object_guid, object_kind,
           display_name, level, is_active
    FROM node_candidates
    WHERE display_name <> ''
    ORDER BY object_kind, object_id, object_guid NULLS LAST, display_name
),
walk AS (
    SELECT n.object_id AS root_object_id, n.object_id AS current_object_id,
           h.parent_object_id, ARRAY[n.display_name]::text[] AS parts,
           ARRAY[n.object_id]::bigint[] AS visited, 0 AS depth
    FROM nodes n
    LEFT JOIN active_hierarchy h ON h.object_id = n.object_id
    UNION ALL
    SELECT w.root_object_id, parent.object_id, h.parent_object_id,
           ARRAY[parent.display_name] || w.parts,
           w.visited || parent.object_id, w.depth + 1
    FROM walk w
    JOIN nodes parent ON parent.object_id = w.parent_object_id
    LEFT JOIN active_hierarchy h ON h.object_id = parent.object_id
    WHERE w.depth < 30 AND NOT parent.object_id = ANY(w.visited)
),
best AS (
    SELECT DISTINCT ON (root_object_id) root_object_id, parts
    FROM walk
    ORDER BY root_object_id, depth DESC
),
ready AS (
    SELECT n.object_id, n.object_guid, h.parent_object_id, n.object_kind,
           n.display_name, array_to_string(b.parts, ', ') AS full_address,
           lower(regexp_replace(regexp_replace(array_to_string(b.parts, ' '), '[[:punct:]]+', ' ', 'g'), '[[:space:]]+', ' ', 'g')) AS search_text,
           n.level, n.is_active
    FROM nodes n
    JOIN best b ON b.root_object_id = n.object_id
    LEFT JOIN active_hierarchy h ON h.object_id = n.object_id
    WHERE n.display_name <> ''
),
ranked AS (
    SELECT ready.*, ROW_NUMBER() OVER (
        PARTITION BY object_kind, lower(full_address)
        ORDER BY object_guid NULLS LAST, object_id
    ) AS duplicate_rank
    FROM ready
)
INSERT INTO gar_search_addresses(object_id, object_guid, parent_object_id, object_kind, display_name, full_address, search_text, level, is_active)
SELECT object_id, object_guid, parent_object_id, object_kind, display_name, full_address, search_text, level, is_active
FROM ranked
WHERE duplicate_rank = 1
ON CONFLICT(object_id) DO UPDATE SET
    object_guid = EXCLUDED.object_guid,
    parent_object_id = EXCLUDED.parent_object_id,
    object_kind = EXCLUDED.object_kind,
    display_name = EXCLUDED.display_name,
    full_address = EXCLUDED.full_address,
    search_text = EXCLUDED.search_text,
    level = EXCLUDED.level,
    is_active = EXCLUDED.is_active;

CREATE UNIQUE INDEX IF NOT EXISTS gar_search_addresses_kind_address_uq
    ON gar_search_addresses (object_kind, lower(full_address));
