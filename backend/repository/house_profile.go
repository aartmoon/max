package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"tvoydom/domain"
)

const selectHouseProfile = `SELECT house_id::text,gis_house_guid::text,gis_house_type,cadastral_number,total_area,living_area,floors,entrances,apartments,year_built,organization,manager,contact,raw_payload,fetched_at FROM gis_house_profiles WHERE house_id=$1`

const upsertHouseProfile = `INSERT INTO gis_house_profiles(house_id,gis_house_guid,gis_house_type,cadastral_number,total_area,living_area,floors,entrances,apartments,year_built,organization,manager,contact,raw_payload,fetched_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (house_id) DO UPDATE SET gis_house_guid=EXCLUDED.gis_house_guid,gis_house_type=EXCLUDED.gis_house_type,cadastral_number=EXCLUDED.cadastral_number,total_area=EXCLUDED.total_area,living_area=EXCLUDED.living_area,floors=EXCLUDED.floors,entrances=EXCLUDED.entrances,apartments=EXCLUDED.apartments,year_built=EXCLUDED.year_built,organization=EXCLUDED.organization,manager=EXCLUDED.manager,contact=EXCLUDED.contact,raw_payload=EXCLUDED.raw_payload,fetched_at=EXCLUDED.fetched_at,updated_at=now()`

type profileTxKey struct{}

func (p Postgres) GetHouseProfile(ctx context.Context, houseID string) (domain.HouseProfile, error) {
	queryer := interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}(p.Pool)
	if tx, ok := ctx.Value(profileTxKey{}).(pgx.Tx); ok {
		queryer = tx
	}
	var profile domain.HouseProfile
	err := queryer.QueryRow(ctx, selectHouseProfile, houseID).Scan(
		&profile.HouseID, &profile.GISHouseGUID, &profile.GISHouseType, &profile.CadastralNumber,
		&profile.TotalArea, &profile.LivingArea, &profile.Floors, &profile.Entrances,
		&profile.Apartments, &profile.YearBuilt, &profile.Organization, &profile.Manager,
		&profile.Contact, &profile.RawPayload, &profile.FetchedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.HouseProfile{}, domain.ErrNotFound
	}
	return profile, err
}

func (p Postgres) SaveHouseProfile(ctx context.Context, profile domain.HouseProfile) error {
	execer := interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	}(p.Pool)
	if tx, ok := ctx.Value(profileTxKey{}).(pgx.Tx); ok {
		execer = tx
	}
	_, err := execer.Exec(ctx, upsertHouseProfile,
		profile.HouseID, profile.GISHouseGUID, profile.GISHouseType, profile.CadastralNumber,
		profile.TotalArea, profile.LivingArea, profile.Floors, profile.Entrances,
		profile.Apartments, profile.YearBuilt, profile.Organization, profile.Manager,
		profile.Contact, profile.RawPayload, profile.FetchedAt,
	)
	return err
}

func (p Postgres) WithHouseProfileLock(ctx context.Context, houseID string, fn func(context.Context) error) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, houseID); err != nil {
		return err
	}
	if err = fn(context.WithValue(ctx, profileTxKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
