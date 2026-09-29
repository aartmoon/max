package repository

import (
	"context"

	"tvoydom/domain"
	"tvoydom/service"
)

func (p Postgres) FindActiveRules(ctx context.Context, q service.RouteQuery) ([]domain.ResponsibilityRule, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT r.id::text,r.house_id::text,r.organization_id::text,o.name,r.category,r.place,r.urgency,
		       r.organization_role,r.valid_from,r.valid_to,r.active,r.source,r.source_url,r.is_demo
		FROM house_responsibility_rules r JOIN organizations o ON o.id=r.organization_id
		WHERE r.house_id=$1 AND r.category=$2 AND r.active
		  AND r.valid_from<=$5 AND (r.valid_to IS NULL OR r.valid_to>$5)
		  AND (r.place='' OR r.place=$3) AND (r.urgency='' OR r.urgency=$4)
		ORDER BY r.id`, q.HouseID, q.Category, q.Place, q.Urgency, q.At.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.ResponsibilityRule{}
	for rows.Next() {
		var item domain.ResponsibilityRule
		if err := rows.Scan(&item.ID, &item.HouseID, &item.OrganizationID, &item.OrganizationName, &item.Category, &item.Place, &item.Urgency, &item.Role, &item.ValidFrom, &item.ValidTo, &item.Active, &item.Source, &item.SourceURL, &item.IsDemo); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
