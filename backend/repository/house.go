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
func (p Postgres) MyHouse(ctx context.Context) (domain.House, error) {
	var h domain.House
	err := p.Pool.QueryRow(ctx, `SELECT h.id::text,h.address,h.total_area,h.living_area,h.floors,h.entrances,h.apartments,h.year_built,COALESCE(o.name,''),h.manager,h.contact FROM houses h LEFT JOIN organizations o ON o.id=h.organization_id WHERE h.address=$1`, "г. Москва, ул. Тестовая, д. 1").Scan(&h.ID, &h.Address, &h.TotalArea, &h.LivingArea, &h.Floors, &h.Entrances, &h.Apartments, &h.YearBuilt, &h.Organization, &h.Manager, &h.Contact)
	return h, err
}
