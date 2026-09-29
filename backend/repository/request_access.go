package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
)

func (p Postgres) RequestAccess(ctx context.Context, id string) (domain.RequestAccess, error) {
	var a domain.RequestAccess
	err := p.Pool.QueryRow(ctx, `
		SELECT r.id::text,r.user_id::text,COALESCE(r.primary_organization_id::text,''),
		       COALESCE(r.contractor_organization_id::text,''),COALESCE(r.assigned_user_id::text,''),
		       COALESCE(uo.organization_id::text,''),r.status
		FROM requests r LEFT JOIN user_organizations uo ON uo.user_id=r.assigned_user_id
		WHERE r.id=$1`, id).Scan(&a.RequestID, &a.OwnerUserID, &a.PrimaryOrganizationID, &a.ContractorOrganizationID, &a.AssigneeUserID, &a.AssigneeOrganizationID, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, domain.ErrNotFound
	}
	return a, err
}

func (p Postgres) ListAccessibleRequests(ctx context.Context, organizationID, userID, queue string, dayStart, dayEnd time.Time) ([]domain.Request, error) {
	rows, err := p.Pool.Query(ctx, selectRequest+`
		WHERE ($1='' OR r.primary_organization_id::text=$1 OR r.contractor_organization_id::text=$1 OR EXISTS(SELECT 1 FROM user_organizations access_uo WHERE access_uo.user_id=r.assigned_user_id AND access_uo.organization_id::text=$1))
		AND CASE $3
		 WHEN 'UNASSIGNED' THEN r.assigned_user_id IS NULL AND r.status NOT IN ('CLOSED','REJECTED')
		 WHEN 'MINE' THEN r.assigned_user_id::text=$2 AND r.status NOT IN ('CLOSED','REJECTED')
		 WHEN 'VISIT_TODAY' THEN r.visit_start >= $4 AND r.visit_start < $5 AND r.status NOT IN ('CLOSED','REJECTED')
		 WHEN 'ALL' THEN true
		 ELSE r.status NOT IN ('CLOSED','REJECTED') END
		ORDER BY r.created_at DESC,r.id DESC`, organizationID, userID, queue, dayStart.UTC(), dayEnd.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Request{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
