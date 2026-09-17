package repository

import (
	"context"
	"tvoydom/domain"
)

// Empty user scope is reserved for the unauthenticated demo organization console.
// A real deployment must supply an authenticated organization scope instead.
func (p Postgres) Organizations(ctx context.Context) ([]domain.Organization, error) {
	rows, err := p.Pool.Query(ctx, `SELECT id::text,name FROM organizations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Organization{}
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}
