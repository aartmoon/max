package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
	"tvoydom/service"
)

func (p Postgres) SaveOrganization(ctx context.Context, id string, in service.SaveOrganizationInput) (domain.Organization, error) {
	var row pgx.Row
	if id == "" {
		row = p.Pool.QueryRow(ctx, `INSERT INTO organizations(id,name,type_code,inn,ogrn,phone,website,active) SELECT COALESCE(max(id),0)+1,$1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7 FROM organizations RETURNING id::text,name,type_code,COALESCE(inn,''),COALESCE(ogrn,''),COALESCE(external_guid,''),phone,website,source,source_url,source_snapshot_at,active`, in.Name, in.Type, in.INN, in.OGRN, in.Phone, in.Website, in.Active)
	} else {
		row = p.Pool.QueryRow(ctx, `UPDATE organizations SET name=$1,type_code=$2,inn=NULLIF($3,''),ogrn=NULLIF($4,''),phone=$5,website=$6,active=$7 WHERE id=$8 RETURNING id::text,name,type_code,COALESCE(inn,''),COALESCE(ogrn,''),COALESCE(external_guid,''),phone,website,source,source_url,source_snapshot_at,active`, in.Name, in.Type, in.INN, in.OGRN, in.Phone, in.Website, in.Active, id)
	}
	return scanOrganization(row)
}

func scanOrganization(row pgx.Row) (o domain.Organization, err error) {
	err = row.Scan(&o.ID, &o.Name, &o.Type, &o.INN, &o.OGRN, &o.ExternalGUID, &o.Phone, &o.Website, &o.Source, &o.SourceURL, &o.SourceSnapshotAt, &o.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return
}

func (p Postgres) SaveResponsibilityRule(ctx context.Context, id string, in service.SaveRuleInput) (domain.ResponsibilityRule, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.ResponsibilityRule{}, err
	}
	defer tx.Rollback(ctx)
	if in.Active && in.Role == "PRIMARY" {
		var conflict bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM house_responsibility_rules WHERE ($1='' OR id<>$1::bigint) AND house_id=$2 AND category=$3 AND place=$4 AND urgency=$5 AND organization_role='PRIMARY' AND active AND tstzrange(valid_from,COALESCE(valid_to,'infinity'),'[)') && tstzrange($6,COALESCE($7,'infinity'),'[)'))`, id, in.HouseID, in.Category, in.Place, in.Urgency, in.ValidFrom, in.ValidTo).Scan(&conflict)
		if err != nil {
			return domain.ResponsibilityRule{}, err
		}
		if conflict {
			return domain.ResponsibilityRule{}, domain.ErrConflict
		}
	}
	var row pgx.Row
	if id == "" {
		row = tx.QueryRow(ctx, `INSERT INTO house_responsibility_rules(house_id,organization_id,category,place,urgency,organization_role,valid_from,valid_to,active,source,source_url,is_demo) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id::text`, in.HouseID, in.OrganizationID, in.Category, in.Place, in.Urgency, in.Role, in.ValidFrom, in.ValidTo, in.Active, in.Source, in.SourceURL, in.IsDemo)
	} else {
		row = tx.QueryRow(ctx, `UPDATE house_responsibility_rules SET house_id=$1,organization_id=$2,category=$3,place=$4,urgency=$5,organization_role=$6,valid_from=$7,valid_to=$8,active=$9,source=$10,source_url=$11,is_demo=$12,updated_at=now() WHERE id=$13 RETURNING id::text`, in.HouseID, in.OrganizationID, in.Category, in.Place, in.Urgency, in.Role, in.ValidFrom, in.ValidTo, in.Active, in.Source, in.SourceURL, in.IsDemo, id)
	}
	var resultID string
	if err = row.Scan(&resultID); errors.Is(err, pgx.ErrNoRows) {
		return domain.ResponsibilityRule{}, domain.ErrNotFound
	} else if err != nil {
		return domain.ResponsibilityRule{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ResponsibilityRule{}, err
	}
	return p.GetResponsibilityRule(ctx, resultID)
}

func (p Postgres) GetResponsibilityRule(ctx context.Context, id string) (domain.ResponsibilityRule, error) {
	var r domain.ResponsibilityRule
	err := p.Pool.QueryRow(ctx, `SELECT r.id::text,r.house_id::text,r.organization_id::text,o.name,r.category,r.place,r.urgency,r.organization_role,r.valid_from,r.valid_to,r.active,r.source,r.source_url,r.is_demo FROM house_responsibility_rules r JOIN organizations o ON o.id=r.organization_id WHERE r.id=$1`, id).Scan(&r.ID, &r.HouseID, &r.OrganizationID, &r.OrganizationName, &r.Category, &r.Place, &r.Urgency, &r.Role, &r.ValidFrom, &r.ValidTo, &r.Active, &r.Source, &r.SourceURL, &r.IsDemo)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, domain.ErrNotFound
	}
	return r, err
}

func (p Postgres) ResponsibilityRules(ctx context.Context) ([]domain.ResponsibilityRule, error) {
	rows, err := p.Pool.Query(ctx, `SELECT r.id::text,r.house_id::text,r.organization_id::text,o.name,r.category,r.place,r.urgency,r.organization_role,r.valid_from,r.valid_to,r.active,r.source,r.source_url,r.is_demo FROM house_responsibility_rules r JOIN organizations o ON o.id=r.organization_id ORDER BY r.house_id,r.category,r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ResponsibilityRule{}
	for rows.Next() {
		var r domain.ResponsibilityRule
		if err := rows.Scan(&r.ID, &r.HouseID, &r.OrganizationID, &r.OrganizationName, &r.Category, &r.Place, &r.Urgency, &r.Role, &r.ValidFrom, &r.ValidTo, &r.Active, &r.Source, &r.SourceURL, &r.IsDemo); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (p Postgres) OrganizationTypes(ctx context.Context) ([]map[string]string, error) {
	rows, err := p.Pool.Query(ctx, `SELECT code,name FROM organization_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		items = append(items, map[string]string{"code": code, "name": name})
	}
	return items, rows.Err()
}

func (p Postgres) ListHouses(ctx context.Context) ([]domain.House, error) {
	rows, err := p.Pool.Query(ctx, `SELECT id::text,COALESCE(gar_object_id::text,''),COALESCE(object_guid::text,''),address FROM houses ORDER BY address`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.House{}
	for rows.Next() {
		var h domain.House
		if err := rows.Scan(&h.ID, &h.GARObjectID, &h.ObjectGUID, &h.Address); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

func (p Postgres) UsersByOrganization(ctx context.Context, organizationID string) ([]domain.User, error) {
	rows, err := p.Pool.Query(ctx, `SELECT u.id::text,COALESCE(u.email,''),u.name FROM users u JOIN user_organizations uo ON uo.user_id=u.id JOIN user_roles ur ON ur.user_id=u.id AND ur.role IN ('manager','admin') WHERE uo.organization_id=$1 ORDER BY u.name`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name); err != nil {
			return nil, err
		}
		u.OrganizationID = &organizationID
		items = append(items, u)
	}
	return items, rows.Err()
}
