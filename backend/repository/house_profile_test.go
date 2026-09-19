package repository

import (
	"strings"
	"testing"

	"tvoydom/domain"
)

func TestHouseProfileMigration(t *testing.T) {
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS gis_house_profiles",
		"house_id bigint PRIMARY KEY REFERENCES houses(id)",
		"raw_payload jsonb NOT NULL",
		"fetched_at timestamptz NOT NULL",
	} {
		if !strings.Contains(gisHouseProfilesMigration, fragment) {
			t.Fatalf("profile migration missing %q", fragment)
		}
	}
}

func TestGARSearchDedupMigration(t *testing.T) {
	for _, fragment := range []string{
		"PARTITION BY object_kind, lower(full_address)",
		"DELETE FROM gar_search_addresses",
		"gar_search_addresses_kind_address_uq",
	} {
		if !strings.Contains(garSearchDedupMigration, fragment) {
			t.Fatalf("GAR search migration missing %q", fragment)
		}
	}
}

func TestHouseProfileQueriesKeepNullableAndRawFields(t *testing.T) {
	for _, fragment := range []string{"cadastral_number", "total_area", "living_area", "floors", "entrances", "apartments", "year_built", "organization", "manager", "contact", "raw_payload", "fetched_at"} {
		if !strings.Contains(selectHouseProfile, fragment) || !strings.Contains(upsertHouseProfile, fragment) {
			t.Fatalf("profile SQL missing %q", fragment)
		}
	}
	if !strings.Contains(upsertHouseProfile, "ON CONFLICT (house_id) DO UPDATE") {
		t.Fatal("profile upsert must replace existing cache row")
	}
}

func TestEnrichedHouseProfileMigrationAndQueries(t *testing.T) {
	columns := []string{
		"house_type_name", "house_status", "project_series", "house_condition",
		"lifecycle_stage", "operation_year", "reconstruction_year",
		"deterioration_percent", "deterioration_date", "wall_material",
		"energy_efficiency", "non_residential_area", "residential_premises",
		"residential_premises_area", "residential_premises_with_realty",
		"residential_premises_with_realty_area", "non_residential_premises",
		"non_residential_premises_area", "non_residential_premises_not_common",
		"non_residential_premises_not_common_area", "owners_or_shares",
		"management_method", "management_organization_guid", "management_short_name",
		"management_full_name", "management_address", "management_phone",
		"management_website", "management_organization_type",
		"management_registry_organization_guid", "management_inn", "management_ogrn",
		"management_chief", "management_contract_start", "management_contract_end",
		"square_payload", "square_summary_available",
	}
	if strings.TrimSpace(enrichedGISHouseProfilesMigration) == "" {
		t.Fatal("enriched migration must be embedded")
	}
	for _, column := range columns {
		if !strings.Contains(enrichedGISHouseProfilesMigration, column) {
			t.Errorf("migration missing %q", column)
		}
		if !strings.Contains(selectHouseProfile, column) {
			t.Errorf("select query missing %q", column)
		}
		if !strings.Contains(upsertHouseProfile, column) {
			t.Errorf("upsert query missing %q", column)
		}
	}
	if !strings.Contains(enrichedGISHouseProfilesMigration, "ADD COLUMN IF NOT EXISTS") {
		t.Fatal("migration must be safe to retry")
	}
}

func TestNormalizeLoadedProfileKeepsOperationYearSeparate(t *testing.T) {
	year := 1980
	profile := domain.HouseProfile{
		YearBuilt:       &year,
		Characteristics: domain.HouseCharacteristics{OperationYear: &year},
	}
	normalizeLoadedProfile(&profile)
	if profile.Characteristics.YearBuilt != nil {
		t.Fatalf("construction year must stay unpublished: %+v", profile.Characteristics)
	}
}
