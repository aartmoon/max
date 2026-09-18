\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;

WITH RECURSIVE candidates AS (
    SELECT DISTINCT object_id
    FROM gar_address_objects
    WHERE lower(name) = lower('Арбат')
      AND level = 8 AND is_actual IS TRUE AND is_active IS TRUE
), ancestry(root_object_id, object_id, depth, visited) AS (
    SELECT object_id, object_id, 0, ARRAY[object_id]::bigint[] FROM candidates
    UNION ALL
    SELECT a.root_object_id, h.parent_object_id, a.depth + 1, a.visited || h.parent_object_id
    FROM ancestry a
    JOIN LATERAL (
        SELECT parent_object_id
        FROM gar_adm_hierarchy
        WHERE object_id = a.object_id AND is_active IS TRUE
        ORDER BY update_date DESC NULLS LAST, id DESC NULLS LAST
        LIMIT 1
    ) h ON h.parent_object_id IS NOT NULL
    WHERE a.depth < 30 AND NOT h.parent_object_id = ANY(a.visited)
), selected_street AS (
    SELECT DISTINCT a.root_object_id AS object_id
    FROM ancestry a
    JOIN gar_address_objects city ON city.object_id = a.object_id
    WHERE lower(city.name) = lower('Москва')
      AND city.level = 1 AND city.is_actual IS TRUE AND city.is_active IS TRUE
)
SELECT count(*) = 1 AS street_ok, min(object_id) AS street_id FROM selected_street \gset
\if :street_ok
\else
  \echo 'expected exactly one active Moscow Arbat street'
  SELECT 1/0 AS abort_export;
\endif

WITH selected_houses AS (
    SELECT DISTINCT unnest(string_to_array(:'house_ids', ',')::bigint[]) AS object_id
)
SELECT count(*) = 10 AS house_input_ok FROM selected_houses \gset
\if :house_input_ok
\else
  \echo 'expected exactly 10 distinct house IDs'
  SELECT 1/0 AS abort_export;
\endif

WITH selected_houses AS (
    SELECT DISTINCT unnest(string_to_array(:'house_ids', ',')::bigint[]) AS object_id
)
SELECT count(DISTINCT chosen.object_id) = 10 AS houses_ok
FROM selected_houses chosen
JOIN gar_houses house ON house.object_id = chosen.object_id
  AND house.is_actual IS TRUE AND house.is_active IS TRUE
JOIN LATERAL (
    SELECT parent_object_id
    FROM gar_adm_hierarchy
    WHERE object_id = chosen.object_id AND is_active IS TRUE
    ORDER BY update_date DESC NULLS LAST, id DESC NULLS LAST
    LIMIT 1
) hierarchy ON hierarchy.parent_object_id = :'street_id'::bigint
\gset
\if :houses_ok
\else
  \echo 'all 10 houses must be active direct children of Moscow Arbat'
  SELECT 1/0 AS abort_export;
\endif

WITH RECURSIVE selected_houses AS (
    SELECT DISTINCT unnest(string_to_array(:'house_ids', ',')::bigint[]) AS object_id
), descendants(root_house, object_id, depth, visited) AS (
    SELECT object_id, object_id, 0, ARRAY[object_id]::bigint[] FROM selected_houses
    UNION ALL
    SELECT d.root_house, h.object_id, d.depth + 1, d.visited || h.object_id
    FROM descendants d
    JOIN gar_adm_hierarchy h ON h.parent_object_id = d.object_id AND h.is_active IS TRUE
    WHERE d.depth < 4 AND NOT h.object_id = ANY(d.visited)
), apartment_objects AS (
    SELECT DISTINCT object_id FROM gar_apartments WHERE is_actual IS TRUE AND is_active IS TRUE
), apartment_ranked AS (
    SELECT d.root_house, a.object_id,
           row_number() OVER (PARTITION BY d.root_house ORDER BY a.object_id) AS rank
    FROM (SELECT DISTINCT root_house, object_id FROM descendants) d
    JOIN apartment_objects a ON a.object_id = d.object_id
), room_objects AS (
    SELECT DISTINCT object_id FROM gar_rooms WHERE is_actual IS TRUE AND is_active IS TRUE
), room_ranked AS (
    SELECT d.root_house, r.object_id,
           row_number() OVER (PARTITION BY d.root_house ORDER BY r.object_id) AS rank
    FROM (SELECT DISTINCT root_house, object_id FROM descendants) d
    JOIN room_objects r ON r.object_id = d.object_id
), carplace_objects AS (
    SELECT DISTINCT object_id FROM gar_carplaces WHERE is_actual IS TRUE AND is_active IS TRUE
), carplace_ranked AS (
    SELECT d.root_house, c.object_id,
           row_number() OVER (PARTITION BY d.root_house ORDER BY c.object_id) AS rank
    FROM (SELECT DISTINCT root_house, object_id FROM descendants) d
    JOIN carplace_objects c ON c.object_id = d.object_id
), stead_ranked AS (
    SELECT DISTINCT s.object_id, row_number() OVER (ORDER BY s.object_id) AS rank
    FROM gar_adm_hierarchy h
    JOIN gar_steads s ON s.object_id = h.object_id
    WHERE h.parent_object_id = :'street_id'::bigint
      AND h.is_active IS TRUE AND s.is_actual IS TRUE AND s.is_active IS TRUE
), selected_base AS (
    SELECT :'street_id'::bigint AS object_id
    UNION SELECT object_id FROM selected_houses
    UNION SELECT object_id FROM apartment_ranked WHERE rank <= 5
    UNION SELECT object_id FROM room_ranked WHERE rank <= 2
    UNION SELECT object_id FROM carplace_ranked WHERE rank <= 2
    UNION SELECT object_id FROM stead_ranked WHERE rank <= 2
), ancestors(object_id, depth, visited) AS (
    SELECT object_id, 0, ARRAY[object_id]::bigint[] FROM selected_base
    UNION ALL
    SELECT h.parent_object_id, a.depth + 1, a.visited || h.parent_object_id
    FROM ancestors a
    JOIN LATERAL (
        SELECT parent_object_id
        FROM gar_adm_hierarchy
        WHERE object_id = a.object_id AND is_active IS TRUE
        ORDER BY update_date DESC NULLS LAST, id DESC NULLS LAST
        LIMIT 1
    ) h ON h.parent_object_id IS NOT NULL
    WHERE a.depth < 30 AND NOT h.parent_object_id = ANY(a.visited)
), selected_ids AS (
    SELECT DISTINCT object_id FROM ancestors
)
SELECT string_agg(object_id::text, ',' ORDER BY object_id) AS selected_ids FROM selected_ids \gset

COPY (SELECT id,object_id,object_guid,change_id,name,type_name,level,operation_type_id,previous_id,next_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_address_objects WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'address_objects_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_addr_object_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'addr_object_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,parent_id,child_id,change_id,raw_attributes FROM gar_addr_object_divisions WHERE parent_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) AND child_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY parent_id,child_id,change_id,id) TO :'addr_object_divisions_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,parent_object_id,change_id,region_code,area_code,city_code,place_code,plan_code,street_code,path,update_date,start_date,end_date,is_active,raw_attributes FROM gar_adm_hierarchy WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'adm_hierarchy_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,parent_object_id,change_id,region_code,area_code,city_code,place_code,plan_code,street_code,path,update_date,start_date,end_date,is_active,oktmo,raw_attributes FROM gar_mun_hierarchy WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'mun_hierarchy_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,object_guid,change_id,house_num,build_num,struc_num,house_type,add_type1,add_num1,add_type2,add_num2,operation_type_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_houses WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'houses_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_house_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'house_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,object_guid,change_id,number,apart_number,apart_type,operation_type_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_apartments WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'apartments_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_apartment_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'apartment_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,object_guid,change_id,number,operation_type_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_carplaces WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'carplaces_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_carplace_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'carplace_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,object_guid,change_id,number,room_type,operation_type_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_rooms WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'rooms_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_room_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'room_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,object_guid,change_id,number,operation_type_id,update_date,start_date,end_date,is_actual,is_active,raw_attributes FROM gar_steads WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,id) TO :'steads_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,object_id,change_id,change_id_end,type_id,value,update_date,start_date,end_date,raw_attributes FROM gar_stead_params WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,type_id,change_id,id) TO :'stead_params_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT change_id,object_id,address_object_id,operation_type_id,normative_doc_id,change_date,raw_attributes FROM gar_change_history WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id,address_object_id) TO :'change_history_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT id,name,number,document_date,document_type,document_kind,update_date,organization_name,registration_number,registration_date,acceptance_date,comment,raw_attributes FROM gar_normative_docs WHERE id IN (SELECT normative_doc_id FROM gar_change_history WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) AND normative_doc_id IS NOT NULL) ORDER BY id) TO :'normative_docs_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT object_id,object_guid,change_id,level_id,create_date,update_date,is_active,raw_attributes FROM gar_reestr_objects WHERE object_id = ANY(string_to_array(:'selected_ids', ',')::bigint[]) ORDER BY object_id,change_id) TO :'reestr_objects_file' WITH (FORMAT csv, HEADER true);
COPY (SELECT current_database() AS database,COALESCE((SELECT source_date::text FROM gar_import_metadata WHERE singleton),'') AS source_date,:'street_id'::bigint AS street_id,replace(:'house_ids', ',', '|') AS house_ids) TO :'source_meta_file' WITH (FORMAT csv, HEADER true);
COMMIT;
