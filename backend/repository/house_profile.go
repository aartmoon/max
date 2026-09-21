package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"tvoydom/domain"
)

const profileColumns = `house_id,gis_house_guid,gis_house_type,cadastral_number,total_area,living_area,floors,entrances,apartments,year_built,organization,manager,contact,
house_type_name,house_status,project_series,house_condition,lifecycle_stage,operation_year,reconstruction_year,deterioration_percent,deterioration_date,wall_material,energy_efficiency,
non_residential_area,residential_premises,residential_premises_area,residential_premises_with_realty,residential_premises_with_realty_area,non_residential_premises,non_residential_premises_area,non_residential_premises_not_common,non_residential_premises_not_common_area,owners_or_shares,
management_method,management_organization_guid,management_short_name,management_full_name,management_address,management_phone,management_website,management_organization_type,management_registry_organization_guid,management_inn,management_ogrn,management_chief,management_contract_start,management_contract_end,
raw_payload,square_payload,square_summary_available,fetched_at`

const selectHouseProfile = `SELECT house_id::text,gis_house_guid::text,gis_house_type,cadastral_number,total_area,living_area,floors,entrances,apartments,year_built,organization,manager,contact,
house_type_name,house_status,project_series,house_condition,lifecycle_stage,operation_year,reconstruction_year,deterioration_percent,deterioration_date,wall_material,energy_efficiency,
non_residential_area,residential_premises,residential_premises_area,residential_premises_with_realty,residential_premises_with_realty_area,non_residential_premises,non_residential_premises_area,non_residential_premises_not_common,non_residential_premises_not_common_area,owners_or_shares,
management_method,management_organization_guid,management_short_name,management_full_name,management_address,management_phone,management_website,management_organization_type,management_registry_organization_guid,management_inn,management_ogrn,management_chief,management_contract_start,management_contract_end,
raw_payload,square_payload,square_summary_available,fetched_at FROM gis_house_profiles WHERE house_id=$1`

const upsertHouseProfile = `INSERT INTO gis_house_profiles(` + profileColumns + `)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52)
ON CONFLICT (house_id) DO UPDATE SET
gis_house_guid=EXCLUDED.gis_house_guid,gis_house_type=EXCLUDED.gis_house_type,cadastral_number=EXCLUDED.cadastral_number,total_area=EXCLUDED.total_area,living_area=EXCLUDED.living_area,floors=EXCLUDED.floors,entrances=EXCLUDED.entrances,apartments=EXCLUDED.apartments,year_built=EXCLUDED.year_built,organization=EXCLUDED.organization,manager=EXCLUDED.manager,contact=EXCLUDED.contact,
house_type_name=EXCLUDED.house_type_name,house_status=EXCLUDED.house_status,project_series=EXCLUDED.project_series,house_condition=EXCLUDED.house_condition,lifecycle_stage=EXCLUDED.lifecycle_stage,operation_year=EXCLUDED.operation_year,reconstruction_year=EXCLUDED.reconstruction_year,deterioration_percent=EXCLUDED.deterioration_percent,deterioration_date=EXCLUDED.deterioration_date,wall_material=EXCLUDED.wall_material,energy_efficiency=EXCLUDED.energy_efficiency,
non_residential_area=EXCLUDED.non_residential_area,residential_premises=EXCLUDED.residential_premises,residential_premises_area=EXCLUDED.residential_premises_area,residential_premises_with_realty=EXCLUDED.residential_premises_with_realty,residential_premises_with_realty_area=EXCLUDED.residential_premises_with_realty_area,non_residential_premises=EXCLUDED.non_residential_premises,non_residential_premises_area=EXCLUDED.non_residential_premises_area,non_residential_premises_not_common=EXCLUDED.non_residential_premises_not_common,non_residential_premises_not_common_area=EXCLUDED.non_residential_premises_not_common_area,owners_or_shares=EXCLUDED.owners_or_shares,
management_method=EXCLUDED.management_method,management_organization_guid=EXCLUDED.management_organization_guid,management_short_name=EXCLUDED.management_short_name,management_full_name=EXCLUDED.management_full_name,management_address=EXCLUDED.management_address,management_phone=EXCLUDED.management_phone,management_website=EXCLUDED.management_website,management_organization_type=EXCLUDED.management_organization_type,management_registry_organization_guid=EXCLUDED.management_registry_organization_guid,management_inn=EXCLUDED.management_inn,management_ogrn=EXCLUDED.management_ogrn,management_chief=EXCLUDED.management_chief,management_contract_start=EXCLUDED.management_contract_start,management_contract_end=EXCLUDED.management_contract_end,
raw_payload=EXCLUDED.raw_payload,square_payload=EXCLUDED.square_payload,square_summary_available=EXCLUDED.square_summary_available,fetched_at=EXCLUDED.fetched_at,updated_at=now()`

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
		&profile.Contact,
		&profile.Characteristics.HouseType, &profile.Characteristics.Status, &profile.Characteristics.ProjectSeries,
		&profile.Characteristics.Condition, &profile.Characteristics.LifecycleStage, &profile.Characteristics.OperationYear,
		&profile.Characteristics.ReconstructionYear, &profile.Characteristics.DeteriorationPercent,
		&profile.Characteristics.DeteriorationDate, &profile.Characteristics.WallMaterial,
		&profile.Characteristics.EnergyEfficiency, &profile.Characteristics.NonResidentialArea,
		&profile.Characteristics.ResidentialPremises, &profile.Characteristics.ResidentialPremisesArea,
		&profile.Characteristics.ResidentialPremisesWithRealty, &profile.Characteristics.ResidentialPremisesWithRealtyArea,
		&profile.Characteristics.NonResidentialPremises, &profile.Characteristics.NonResidentialPremisesArea,
		&profile.Characteristics.NonResidentialPremisesNotCommon, &profile.Characteristics.NonResidentialPremisesNotCommonArea,
		&profile.Characteristics.OwnersOrShares,
		&profile.Management.Method, &profile.Management.OrganizationGUID, &profile.Management.ShortName,
		&profile.Management.FullName, &profile.Management.Address, &profile.Management.Phone,
		&profile.Management.Website, &profile.Management.OrganizationType, &profile.Management.RegistryOrganizationGUID,
		&profile.Management.INN, &profile.Management.OGRN, &profile.Management.Chief,
		&profile.Management.ContractStart, &profile.Management.ContractEnd,
		&profile.RawPayload, &profile.SquarePayload, &profile.SquareSummaryAvailable, &profile.FetchedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.HouseProfile{}, domain.ErrNotFound
	}
	if err != nil {
		return profile, err
	}
	normalizeLoadedProfile(&profile)
	return profile, nil
}

func (p Postgres) SaveHouseProfile(ctx context.Context, profile domain.HouseProfile) error {
	normalizeProfileForSave(&profile)
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
		profile.Contact,
		profile.Characteristics.HouseType, profile.Characteristics.Status, profile.Characteristics.ProjectSeries,
		profile.Characteristics.Condition, profile.Characteristics.LifecycleStage, profile.Characteristics.OperationYear,
		profile.Characteristics.ReconstructionYear, profile.Characteristics.DeteriorationPercent,
		profile.Characteristics.DeteriorationDate, profile.Characteristics.WallMaterial,
		profile.Characteristics.EnergyEfficiency, profile.Characteristics.NonResidentialArea,
		profile.Characteristics.ResidentialPremises, profile.Characteristics.ResidentialPremisesArea,
		profile.Characteristics.ResidentialPremisesWithRealty, profile.Characteristics.ResidentialPremisesWithRealtyArea,
		profile.Characteristics.NonResidentialPremises, profile.Characteristics.NonResidentialPremisesArea,
		profile.Characteristics.NonResidentialPremisesNotCommon, profile.Characteristics.NonResidentialPremisesNotCommonArea,
		profile.Characteristics.OwnersOrShares,
		profile.Management.Method, profile.Management.OrganizationGUID, profile.Management.ShortName,
		profile.Management.FullName, profile.Management.Address, profile.Management.Phone,
		profile.Management.Website, profile.Management.OrganizationType, profile.Management.RegistryOrganizationGUID,
		profile.Management.INN, profile.Management.OGRN, profile.Management.Chief,
		profile.Management.ContractStart, profile.Management.ContractEnd,
		profile.RawPayload, nullableJSON(profile.SquarePayload), profile.SquareSummaryAvailable, profile.FetchedAt,
	)
	return err
}

func normalizeLoadedProfile(profile *domain.HouseProfile) {
	profile.Characteristics.HouseTypeCode = stringFallback(profile.Characteristics.HouseTypeCode, stringPointer(profile.GISHouseType))
	profile.Characteristics.TotalArea = floatFallback(profile.Characteristics.TotalArea, profile.TotalArea)
	profile.Characteristics.LivingArea = floatFallback(profile.Characteristics.LivingArea, profile.LivingArea)
	profile.Characteristics.Floors = intFallback(profile.Characteristics.Floors, profile.Floors)
	profile.Characteristics.Entrances = intFallback(profile.Characteristics.Entrances, profile.Entrances)
	profile.Characteristics.ResidentialPremises = intFallback(profile.Characteristics.ResidentialPremises, profile.Apartments)
	if profile.Characteristics.OperationYear == nil {
		profile.Characteristics.YearBuilt = intFallback(profile.Characteristics.YearBuilt, profile.YearBuilt)
	}
	if profile.Management.ShortName == nil && profile.Management.FullName == nil {
		profile.Management.ShortName = profile.Organization
	}
	profile.Management.Chief = stringFallback(profile.Management.Chief, profile.Manager)
	profile.Management.Phone = stringFallback(profile.Management.Phone, profile.Contact)
	profile.DataSources = []domain.HouseDataSource{
		{Name: "ГИС ЖКХ · карточка дома", Available: true, UpdatedAt: profile.FetchedAt, Stale: profile.Stale},
		{Name: "ГИС ЖКХ · сводка помещений", Available: profile.SquareSummaryAvailable, UpdatedAt: profile.FetchedAt, Stale: profile.Stale},
	}
}

func normalizeProfileForSave(profile *domain.HouseProfile) {
	normalizeLoadedProfile(profile)
	profile.TotalArea = profile.Characteristics.TotalArea
	profile.LivingArea = profile.Characteristics.LivingArea
	profile.Floors = profile.Characteristics.Floors
	profile.Entrances = profile.Characteristics.Entrances
	profile.Apartments = profile.Characteristics.ResidentialPremises
	profile.YearBuilt = intFallback(profile.Characteristics.YearBuilt, profile.Characteristics.OperationYear)
	profile.Organization = stringFallback(profile.Management.ShortName, profile.Management.FullName)
	profile.Manager = profile.Management.Chief
	profile.Contact = profile.Management.Phone
}

func stringFallback(primary, fallback *string) *string {
	if primary != nil {
		return primary
	}
	return fallback
}

func intFallback(primary, fallback *int) *int {
	if primary != nil {
		return primary
	}
	return fallback
}

func floatFallback(primary, fallback *float64) *float64 {
	if primary != nil {
		return primary
	}
	return fallback
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
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
