package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresPostgresSettings(t *testing.T) {
	_, err := Load(func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_HOST") {
		t.Fatalf("expected POSTGRES_HOST error, got %v", err)
	}
}

func TestLoadAppliesGARDefaults(t *testing.T) {
	values := map[string]string{
		"POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432",
		"POSTGRES_DB": "app", "POSTGRES_USER": "app", "POSTGRES_PASSWORD": "secret",
	}
	got, err := Load(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDir != "/data/gar" || got.BatchSize != 10000 || got.LogEvery != 100000 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
	if got.AllowDemoData {
		t.Fatal("demo data must be opt-in")
	}
}

func TestLoadParsesOverridesAndConnectionString(t *testing.T) {
	values := map[string]string{
		"POSTGRES_HOST": "db host", "POSTGRES_PORT": "5434", "POSTGRES_DB": "app/db",
		"POSTGRES_USER": "app user", "POSTGRES_PASSWORD": "p@ss word", "GAR_DATA_DIR": "/fixtures",
		"GAR_BATCH_SIZE": "17", "GAR_LOG_EVERY": "23", "GAR_ALLOW_DEMO_DATA": "true",
	}
	got, err := Load(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if got.BatchSize != 17 || got.LogEvery != 23 || !got.AllowDemoData {
		t.Fatalf("unexpected overrides: %+v", got)
	}
	want := "postgres://app%20user:p%40ss%20word@db%20host:5434/app%2Fdb?sslmode=disable"
	if got.DatabaseURL() != want {
		t.Fatalf("connection string\nwant: %s\n got: %s", want, got.DatabaseURL())
	}
}

func TestLoadRejectsInvalidNumericAndBooleanValues(t *testing.T) {
	base := map[string]string{
		"POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432", "POSTGRES_DB": "app",
		"POSTGRES_USER": "app", "POSTGRES_PASSWORD": "secret",
	}
	for key, value := range map[string]string{
		"POSTGRES_PORT": "x", "GAR_BATCH_SIZE": "0", "GAR_LOG_EVERY": "-1", "GAR_ALLOW_DEMO_DATA": "sometimes",
	} {
		t.Run(key, func(t *testing.T) {
			values := mapsClone(base)
			values[key] = value
			_, err := Load(func(name string) string { return values[name] })
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected %s error, got %v", key, err)
			}
		})
	}
}

func mapsClone(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
