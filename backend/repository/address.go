package repository

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"tvoydom/domain"
	"tvoydom/integration/gar"

	"github.com/jackc/pgx/v5"
)

var searchCleanup = regexp.MustCompile(`[[:punct:]]+`)
var searchSpaces = regexp.MustCompile(`\s+`)

func NormalizeSearchText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = searchCleanup.ReplaceAllString(s, " ")
	return searchSpaces.ReplaceAllString(s, " ")
}

func SearchTokens(s string) []string {
	normalized := NormalizeSearchText(s)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}

func (p Postgres) Search(ctx context.Context, query string, limit int) ([]domain.AddressSuggestion, error) {
	tokens := SearchTokens(query)
	if len(tokens) == 0 {
		return []domain.AddressSuggestion{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	sql := `SELECT fias_guid::text,address,object_type FROM addresses WHERE object_type='house' AND is_active`
	args := []any{}
	for _, token := range tokens {
		args = append(args, "%"+token+"%")
		sql += ` AND search_text LIKE $` + strconv.Itoa(len(args))
	}
	args = append(args, NormalizeSearchText(query), limit)
	sql += ` ORDER BY similarity(search_text,$` + strconv.Itoa(len(args)-1) + `) DESC, address LIMIT $` + strconv.Itoa(len(args))
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.AddressSuggestion{}
	for rows.Next() {
		var item domain.AddressSuggestion
		if err := rows.Scan(&item.FIASGUID, &item.Address, &item.ObjectType); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (p Postgres) GetByGUID(ctx context.Context, fiasGUID string) (*domain.AddressInfo, error) {
	var item domain.AddressInfo
	err := p.Pool.QueryRow(ctx, `
		SELECT fias_guid::text,address,postal_code,cadastral_number,object_type
		FROM addresses
		WHERE fias_guid=$1`, fiasGUID).Scan(&item.FIASGUID, &item.Address, &item.PostalCode, &item.CadastralNumber, &item.ObjectType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (p Postgres) GetHouseByFIASGUID(ctx context.Context, fiasGUID string) (domain.House, error) {
	return p.scanHouse(p.Pool.QueryRow(ctx, `SELECT id::text,COALESCE(fias_guid::text,''),address,cadastral_number,created_at,updated_at,total_area,living_area,floors,entrances,apartments,year_built,'' AS organization,manager,contact FROM houses WHERE fias_guid=$1`, fiasGUID))
}

func (p Postgres) GetHouse(ctx context.Context, id string) (domain.House, error) {
	return p.scanHouse(p.Pool.QueryRow(ctx, `SELECT id::text,COALESCE(fias_guid::text,''),address,cadastral_number,created_at,updated_at,total_area,living_area,floors,entrances,apartments,year_built,'' AS organization,manager,contact FROM houses WHERE id=$1`, id))
}

func (p Postgres) ResolveHouse(ctx context.Context, in domain.HouseIdentity) (domain.House, error) {
	return p.scanHouse(p.Pool.QueryRow(ctx, `
		INSERT INTO houses(fias_guid,address,cadastral_number,updated_at)
		VALUES($1,$2,$3,now())
		ON CONFLICT(fias_guid) WHERE fias_guid IS NOT NULL
		DO UPDATE SET fias_guid=EXCLUDED.fias_guid
		RETURNING id::text,COALESCE(fias_guid::text,''),address,cadastral_number,created_at,updated_at,total_area,living_area,floors,entrances,apartments,year_built,'' AS organization,manager,contact`,
		in.FIASGUID, in.Address, in.CadastralNumber))
}

func (p Postgres) scanHouse(row pgx.Row) (domain.House, error) {
	var h domain.House
	var createdAt, updatedAt time.Time
	err := row.Scan(&h.ID, &h.FIASGUID, &h.Address, &h.CadastralNumber, &createdAt, &updatedAt, &h.TotalArea, &h.LivingArea, &h.Floors, &h.Entrances, &h.Apartments, &h.YearBuilt, &h.Organization, &h.Manager, &h.Contact)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, domain.ErrNotFound
	}
	if err != nil {
		return h, err
	}
	h.CreatedAt = &createdAt
	h.UpdatedAt = &updatedAt
	return h, nil
}

func (p Postgres) UpsertAddressObjects(ctx context.Context, items []gar.AddressObject) error {
	b := &pgx.Batch{}
	for _, item := range items {
		b.Queue(`INSERT INTO gar_address_objects(object_id,object_guid,name,type_name,level,is_active,is_actual,gar_updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT(object_id) DO UPDATE SET object_guid=EXCLUDED.object_guid,name=EXCLUDED.name,type_name=EXCLUDED.type_name,level=EXCLUDED.level,is_active=EXCLUDED.is_active,is_actual=EXCLUDED.is_actual,gar_updated_at=EXCLUDED.gar_updated_at`,
			item.ObjectID, item.ObjectGUID, item.Name, item.TypeName, item.Level, item.IsActive, item.IsActual, nullDate(item.UpdatedAt))
	}
	return execBatch(ctx, p, b)
}

func (p Postgres) UpsertHierarchy(ctx context.Context, items []gar.HierarchyItem) error {
	b := &pgx.Batch{}
	for _, item := range items {
		b.Queue(`INSERT INTO gar_adm_hierarchy(object_id,parent_object_id,is_active,gar_updated_at)
VALUES($1,$2,$3,$4)
ON CONFLICT(object_id) DO UPDATE SET parent_object_id=EXCLUDED.parent_object_id,is_active=EXCLUDED.is_active,gar_updated_at=EXCLUDED.gar_updated_at`,
			item.ObjectID, item.ParentObjectID, item.IsActive, nullDate(item.UpdatedAt))
	}
	return execBatch(ctx, p, b)
}

func (p Postgres) UpsertHouseTypes(ctx context.Context, items []gar.HouseType) error {
	b := &pgx.Batch{}
	for _, item := range items {
		short := item.ShortName
		if short == "" {
			short = item.Name
		}
		b.Queue(`INSERT INTO gar_house_types(type_id,name,short_name,is_active)
VALUES($1,$2,$3,$4)
ON CONFLICT(type_id) DO UPDATE SET name=EXCLUDED.name,short_name=EXCLUDED.short_name,is_active=EXCLUDED.is_active`,
			item.ID, item.Name, short, item.IsActive)
	}
	return execBatch(ctx, p, b)
}

func (p Postgres) UpsertParamTypes(ctx context.Context, items []gar.ParamType) error {
	b := &pgx.Batch{}
	for _, item := range items {
		b.Queue(`INSERT INTO gar_param_types(type_id,name,code,is_active)
VALUES($1,$2,$3,$4)
ON CONFLICT(type_id) DO UPDATE SET name=EXCLUDED.name,code=EXCLUDED.code,is_active=EXCLUDED.is_active`,
			item.ID, item.Name, item.Code, item.IsActive)
	}
	return execBatch(ctx, p, b)
}

func (p Postgres) UpsertParams(ctx context.Context, items []gar.Param) error {
	b := &pgx.Batch{}
	for _, item := range items {
		b.Queue(`INSERT INTO gar_params(object_id,type_id,value,gar_updated_at)
VALUES($1,$2,$3,$4)
ON CONFLICT(object_id,type_id) DO UPDATE SET value=EXCLUDED.value,gar_updated_at=EXCLUDED.gar_updated_at`,
			item.ObjectID, item.TypeID, item.Value, nullDate(item.UpdatedAt))
	}
	return execBatch(ctx, p, b)
}

func (p Postgres) UpsertHouses(ctx context.Context, items []gar.HouseRecord) error {
	b := &pgx.Batch{}
	for _, item := range items {
		b.Queue(`INSERT INTO gar_houses(object_id,object_guid,house_num,add_num1,add_num2,house_type,add_type1,add_type2,is_active,is_actual,gar_updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(object_id) DO UPDATE SET object_guid=EXCLUDED.object_guid,house_num=EXCLUDED.house_num,add_num1=EXCLUDED.add_num1,add_num2=EXCLUDED.add_num2,house_type=EXCLUDED.house_type,add_type1=EXCLUDED.add_type1,add_type2=EXCLUDED.add_type2,is_active=EXCLUDED.is_active,is_actual=EXCLUDED.is_actual,gar_updated_at=EXCLUDED.gar_updated_at`,
			item.ObjectID, item.ObjectGUID, item.HouseNum, item.AddNum1, item.AddNum2, item.HouseType, item.AddType1, item.AddType2, item.IsActive, item.IsActual, nullDate(item.UpdatedAt))
	}
	return execBatch(ctx, p, b)
}

func execBatch(ctx context.Context, p Postgres, b *pgx.Batch) error {
	if b.Len() == 0 {
		return nil
	}
	br := p.Pool.SendBatch(ctx, b)
	defer br.Close()
	for i := 0; i < b.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func nullDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (p Postgres) RefreshAddresses(ctx context.Context) error {
	_, err := p.Pool.Exec(ctx, refreshAddressesSQL)
	return err
}

// refreshAddressesSQL derives postal_code and cadastral_number through
// gar_param_types.CODE/NAME instead of hardcoded TYPEID values. The official
// GAR PARAM row carries TYPEID only; PARAMTYPES is the deterministic semantic
// mapping from TYPEID to parameter kind for this importer.
const refreshAddressesSQL = `
WITH RECURSIVE chain AS (
 SELECT h.object_id AS house_object_id, ah.parent_object_id AS object_id, 1 AS depth
 FROM gar_houses h
 JOIN gar_adm_hierarchy ah ON ah.object_id=h.object_id AND ah.is_active
 WHERE ah.parent_object_id IS NOT NULL
 UNION ALL
 SELECT c.house_object_id, ah.parent_object_id, c.depth+1
 FROM chain c
 JOIN gar_adm_hierarchy ah ON ah.object_id=c.object_id AND ah.is_active
 WHERE ah.parent_object_id IS NOT NULL AND c.depth < 20
),
components AS (
 SELECT c.house_object_id,c.depth,o.object_id,o.type_name,o.name,o.level,(o.type_name || ' ' || o.name) AS label
 FROM chain c
 JOIN gar_address_objects o ON o.object_id=c.object_id
 WHERE o.is_active AND o.is_actual
),
component_text AS (
 SELECT house_object_id,
        string_agg(label, ', ' ORDER BY depth DESC) AS parent_address,
        max(label) FILTER (WHERE level=1) AS region,
        max(label) FILTER (WHERE level IN (2,3)) AS area,
        max(label) FILTER (WHERE lower(type_name) IN ('г','город')) AS city,
        max(label) FILTER (WHERE level IN (5,6) AND lower(type_name) NOT IN ('г','город')) AS settlement,
        max(label) FILTER (WHERE level IN (7,8)) AS street
 FROM components
 GROUP BY house_object_id
),
params AS (
 SELECT p.object_id,
        max(p.value) FILTER (WHERE lower(pt.code || ' ' || pt.name) LIKE '%post%' OR lower(pt.code || ' ' || pt.name) LIKE '%индекс%') AS postal_code,
        max(p.value) FILTER (WHERE lower(pt.code || ' ' || pt.name) LIKE '%cad%' OR lower(pt.code || ' ' || pt.name) LIKE '%кадастр%') AS cadastral_number
 FROM gar_params p
 JOIN gar_param_types pt ON pt.type_id=p.type_id AND pt.is_active
 GROUP BY p.object_id
),
ready AS (
 SELECT h.object_guid AS fias_guid,
        h.object_id AS gar_object_id,
        trim(both ', ' FROM concat_ws(', ', ct.parent_address,
          nullif(trim(concat_ws(' ', ht.short_name, h.house_num)), ''),
          nullif(trim(concat_ws(' ', at1.short_name, h.add_num1)), ''),
          nullif(trim(concat_ws(' ', at2.short_name, h.add_num2)), '')
        )) AS address,
        'house' AS object_type,
        coalesce(ct.region,'') AS region,
        coalesce(ct.area,'') AS area,
        coalesce(ct.city,'') AS city,
        coalesce(ct.settlement,'') AS settlement,
        coalesce(ct.street,'') AS street,
        trim(concat_ws(' ', ht.short_name, h.house_num, at1.short_name, h.add_num1, at2.short_name, h.add_num2)) AS house,
        p.postal_code,
        p.cadastral_number,
        h.is_active AND h.is_actual AS is_active,
        h.gar_updated_at
 FROM gar_houses h
 LEFT JOIN component_text ct ON ct.house_object_id=h.object_id
 LEFT JOIN gar_house_types ht ON ht.type_id=h.house_type
 LEFT JOIN gar_house_types at1 ON at1.type_id=h.add_type1
 LEFT JOIN gar_house_types at2 ON at2.type_id=h.add_type2
 LEFT JOIN params p ON p.object_id=h.object_id
)
INSERT INTO addresses(fias_guid,gar_object_id,address,search_text,object_type,region,area,city,settlement,street,house,postal_code,cadastral_number,is_active,gar_updated_at,imported_at)
SELECT fias_guid,gar_object_id,address,
       lower(regexp_replace(regexp_replace(concat_ws(' ',address,region,area,city,settlement,street,house), '[[:punct:]]+', ' ', 'g'), '[[:space:]]+', ' ', 'g')),
       object_type,region,area,city,settlement,street,house,postal_code,cadastral_number,is_active,gar_updated_at,now()
FROM ready
WHERE address <> ''
ON CONFLICT(fias_guid) DO UPDATE SET
 gar_object_id=EXCLUDED.gar_object_id,
 address=EXCLUDED.address,
 search_text=EXCLUDED.search_text,
 object_type=EXCLUDED.object_type,
 region=EXCLUDED.region,
 area=EXCLUDED.area,
 city=EXCLUDED.city,
 settlement=EXCLUDED.settlement,
 street=EXCLUDED.street,
 house=EXCLUDED.house,
 postal_code=EXCLUDED.postal_code,
 cadastral_number=EXCLUDED.cadastral_number,
 is_active=EXCLUDED.is_active,
 gar_updated_at=EXCLUDED.gar_updated_at,
 imported_at=EXCLUDED.imported_at`
