package repository

import (
	"strings"
	"testing"
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
