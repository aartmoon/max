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

func (p Postgres) CreateOrganization(ctx context.Context, name string) (domain.Organization, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Organization{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE organizations IN EXCLUSIVE MODE`); err != nil {
		return domain.Organization{}, err
	}
	var org domain.Organization
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations(id,name)
		SELECT COALESCE(MAX(id),0)+1,$1 FROM organizations
		RETURNING id::text,name`, name).Scan(&org.ID, &org.Name)
	if err != nil {
		return domain.Organization{}, err
	}
	return org, tx.Commit(ctx)
}
