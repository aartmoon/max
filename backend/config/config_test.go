package config

import (
	"testing"
	"time"
)

func TestGISDefaults(t *testing.T) {
	t.Setenv("GIS_HOUSING_BASE_URL", "")
	t.Setenv("GIS_HOUSING_TIMEOUT", "")
	t.Setenv("GIS_HOUSE_CACHE_TTL", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.GISHousingBaseURL != "https://dom.gosuslugi.ru" || c.GISHousingTimeout != 8*time.Second || c.GISHouseCacheTTL != 24*time.Hour {
		t.Fatalf("unexpected GIS defaults: %+v", c)
	}
}

func TestGISDurationsRejectInvalidValues(t *testing.T) {
	for _, name := range []string{"GIS_HOUSING_TIMEOUT", "GIS_HOUSE_CACHE_TTL"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "not-a-duration")
			if _, err := Load(); err == nil {
				t.Fatal("expected invalid duration error")
			}
		})
	}
}
