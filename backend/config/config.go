package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	DatabaseURL, Port                                     string
	MockStatusEnabled                                     bool
	MaxBotToken, MaxAppURL, MaxBotUsername, MaxCACertFile string
	GISHousingBaseURL                                     string
	GISHousingTimeout, GISHouseCacheTTL                   time.Duration
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL := os.Getenv("GIS_HOUSING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dom.gosuslugi.ru"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return Config{}, fmt.Errorf("GIS_HOUSING_BASE_URL must be an absolute HTTPS URL")
	}
	timeout, err := duration("GIS_HOUSING_TIMEOUT", 8*time.Second)
	if err != nil {
		return Config{}, err
	}
	ttl, err := duration("GIS_HOUSE_CACHE_TTL", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	return Config{MaxBotToken: os.Getenv("MAX_BOT_TOKEN"), MaxAppURL: os.Getenv("MAX_APP_URL"), MaxBotUsername: os.Getenv("MAX_BOT_USERNAME"), MaxCACertFile: os.Getenv("MAX_CA_CERT_FILE"), DatabaseURL: os.Getenv("DATABASE_URL"), Port: port, MockStatusEnabled: os.Getenv("MOCK_STATUS_ENABLED") != "false", GISHousingBaseURL: baseURL, GISHousingTimeout: timeout, GISHouseCacheTTL: ttl}, nil
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}
