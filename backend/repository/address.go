package repository

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
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

func (p Postgres) Search(ctx context.Context, in domain.AddressSearch) ([]domain.AddressSuggestion, error) {
	sql, args := buildAddressSearchQuery(in)
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.AddressSuggestion{}
	for rows.Next() {
		var item domain.AddressSuggestion
		if err := rows.Scan(&item.ObjectID, &item.ObjectGUID, &item.ParentObjectID, &item.ObjectKind, &item.DisplayName, &item.FullAddress); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func buildAddressSearchQuery(in domain.AddressSearch) (string, []any) {
	if in.Limit <= 0 || in.Limit > 50 {
		in.Limit = 10
	}
	sql := `SELECT object_id::text,COALESCE(object_guid::text,''),COALESCE(parent_object_id::text,''),object_kind,display_name,full_address FROM gar_search_addresses WHERE is_active`
	args := []any{}
	if in.Kind != "" {
		args = append(args, in.Kind)
		sql += ` AND object_kind=$` + strconv.Itoa(len(args))
	}
	if in.ParentObjectID != nil {
		args = append(args, *in.ParentObjectID)
		sql += ` AND parent_object_id=$` + strconv.Itoa(len(args))
	}
	for _, token := range SearchTokens(in.Query) {
		args = append(args, "%"+token+"%")
		sql += ` AND search_text LIKE $` + strconv.Itoa(len(args))
	}
	normalized := NormalizeSearchText(in.Query)
	args = append(args, normalized)
	rankArg := len(args)
	args = append(args, in.Limit)
	sql += ` ORDER BY CASE WHEN lower(display_name) LIKE $` + strconv.Itoa(rankArg) + ` || '%' THEN 0 WHEN search_text LIKE $` + strconv.Itoa(rankArg) + ` || '%' THEN 1 ELSE 2 END, similarity(search_text,$` + strconv.Itoa(rankArg) + `) DESC,full_address LIMIT $` + strconv.Itoa(len(args))
	return sql, args
}

func (p Postgres) GetAddress(ctx context.Context, objectID int64) (domain.AddressInfo, error) {
	var item domain.AddressInfo
	err := p.Pool.QueryRow(ctx, `SELECT object_id::text,COALESCE(object_guid::text,''),COALESCE(parent_object_id::text,''),object_kind,display_name,full_address,is_active FROM gar_search_addresses WHERE object_id=$1`, objectID).
		Scan(&item.ObjectID, &item.ObjectGUID, &item.ParentObjectID, &item.ObjectKind, &item.DisplayName, &item.FullAddress, &item.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AddressInfo{}, domain.ErrNotFound
	}
	return item, err
}

func (p Postgres) GetHouseByGARObjectID(ctx context.Context, objectID int64) (domain.House, error) {
	return p.scanHouse(p.Pool.QueryRow(ctx, houseSelect+` WHERE gar_object_id=$1`, objectID))
}

func (p Postgres) GetHouse(ctx context.Context, id string) (domain.House, error) {
	return p.scanHouse(p.Pool.QueryRow(ctx, houseSelect+` WHERE id=$1`, id))
}

func (p Postgres) ResolveHouse(ctx context.Context, info domain.AddressInfo) (domain.House, error) {
	objectID, err := strconv.ParseInt(info.ObjectID, 10, 64)
	if err != nil || objectID <= 0 {
		return domain.House{}, domain.ErrInvalidAddressID
	}
	return p.scanHouse(p.Pool.QueryRow(ctx, `
        INSERT INTO houses(gar_object_id,object_guid,address,updated_at)
        VALUES($1,NULLIF($2,'')::uuid,$3,now())
        ON CONFLICT(gar_object_id) WHERE gar_object_id IS NOT NULL
        DO UPDATE SET object_guid=EXCLUDED.object_guid,address=EXCLUDED.address,updated_at=now()
        RETURNING id::text,gar_object_id::text,COALESCE(object_guid::text,''),address,cadastral_number,created_at,updated_at,total_area,living_area,floors,entrances,apartments,year_built,'' AS organization,manager,contact`,
		objectID, info.ObjectGUID, info.FullAddress))
}

const houseSelect = `SELECT id::text,COALESCE(gar_object_id::text,''),COALESCE(object_guid::text,''),address,cadastral_number,created_at,updated_at,total_area,living_area,floors,entrances,apartments,year_built,'' AS organization,manager,contact FROM houses`

func (p Postgres) scanHouse(row pgx.Row) (domain.House, error) {
	var h domain.House
	var createdAt, updatedAt time.Time
	err := row.Scan(&h.ID, &h.GARObjectID, &h.ObjectGUID, &h.Address, &h.CadastralNumber, &createdAt, &updatedAt, &h.TotalArea, &h.LivingArea, &h.Floors, &h.Entrances, &h.Apartments, &h.YearBuilt, &h.Organization, &h.Manager, &h.Contact)
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
