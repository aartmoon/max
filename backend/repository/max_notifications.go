package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"tvoydom/service"
)

func (p Postgres) RequestOwnerMAXDestination(ctx context.Context, requestID string) (service.MAXDestination, bool, error) {
	var destination service.MAXDestination
	err := p.Pool.QueryRow(ctx, `
		SELECT a.max_user_id,a.chat_id
		FROM requests r JOIN user_max_accounts a ON a.user_id=r.user_id
		WHERE r.id=$1`, requestID).Scan(&destination.UserID, &destination.ChatID)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.MAXDestination{}, false, nil
	}
	return destination, err == nil, err
}

func (p Postgres) ManagerMAXDestinations(ctx context.Context, requestID string) ([]service.MAXDestination, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT DISTINCT a.max_user_id,a.chat_id
		FROM requests r
		JOIN user_organizations uo ON uo.organization_id=r.primary_organization_id OR uo.organization_id=r.contractor_organization_id
		JOIN user_roles ur ON ur.user_id=uo.user_id AND ur.role IN ('manager','admin')
		JOIN user_max_accounts a ON a.user_id=uo.user_id
		WHERE r.id=$1
		ORDER BY a.max_user_id`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.MAXDestination
	for rows.Next() {
		var destination service.MAXDestination
		if err := rows.Scan(&destination.UserID, &destination.ChatID); err != nil {
			return nil, err
		}
		result = append(result, destination)
	}
	return result, rows.Err()
}
