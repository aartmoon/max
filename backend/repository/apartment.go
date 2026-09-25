package repository

import (
	"context"
	"errors"
	"tvoydom/domain"

	"github.com/jackc/pgx/v5"
)

func (p Postgres) UserApartments(ctx context.Context, userID string) ([]domain.UserApartment, error) {
	rows, err := p.Pool.Query(ctx, `SELECT id::text,user_id::text,house_object_id::text,COALESCE(house_object_guid::text,''),COALESCE(apartment_object_id::text,''),COALESCE(apartment_object_guid::text,''),address,label,is_default FROM user_apartments WHERE user_id=$1 ORDER BY is_default DESC,id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.UserApartment
	for rows.Next() {
		var a domain.UserApartment
		if err := rows.Scan(&a.ID, &a.UserID, &a.HouseObjectID, &a.HouseObjectGUID, &a.ApartmentObjectID, &a.ApartmentObjectGUID, &a.Address, &a.Label, &a.IsDefault); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (p Postgres) CreateUserApartment(ctx context.Context, a domain.UserApartment) (domain.UserApartment, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_apartments WHERE user_id=$1)`, a.UserID).Scan(&exists); err != nil {
		return a, err
	}
	if !exists {
		a.IsDefault = true
	}
	if a.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE user_apartments SET is_default=false WHERE user_id=$1`, a.UserID); err != nil {
			return a, err
		}
	}
	err = tx.QueryRow(ctx, `INSERT INTO user_apartments(user_id,house_object_id,house_object_guid,apartment_object_id,apartment_object_guid,address,label,is_default)
VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::bigint,NULLIF($5,'')::uuid,$6,$7,$8)
RETURNING id::text`, a.UserID, a.HouseObjectID, a.HouseObjectGUID, a.ApartmentObjectID, a.ApartmentObjectGUID, a.Address, a.Label, a.IsDefault).Scan(&a.ID)
	if err != nil {
		return a, err
	}
	return a, tx.Commit(ctx)
}

func (p Postgres) SetDefaultApartment(ctx context.Context, userID, id string) (domain.UserApartment, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.UserApartment{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE user_apartments SET is_default=false WHERE user_id=$1`, userID)
	if err != nil {
		return domain.UserApartment{}, err
	}
	_ = tag
	var a domain.UserApartment
	err = tx.QueryRow(ctx, `UPDATE user_apartments SET is_default=true WHERE id=$1 AND user_id=$2 RETURNING id::text,user_id::text,house_object_id::text,COALESCE(house_object_guid::text,''),COALESCE(apartment_object_id::text,''),COALESCE(apartment_object_guid::text,''),address,label,is_default`, id, userID).Scan(&a.ID, &a.UserID, &a.HouseObjectID, &a.HouseObjectGUID, &a.ApartmentObjectID, &a.ApartmentObjectGUID, &a.Address, &a.Label, &a.IsDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserApartment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.UserApartment{}, err
	}
	return a, tx.Commit(ctx)
}

func (p Postgres) DeleteUserApartment(ctx context.Context, userID, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM user_apartments WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
