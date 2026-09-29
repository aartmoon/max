package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL, Port                                     string
	MockStatusEnabled                                     bool
	MaxBotToken, MaxAppURL, MaxBotUsername, MaxCACertFile string
	GISHousingBaseURL                                     string
	GISHousingTimeout, GISHouseCacheTTL                   time.Duration
	SMTPHost, SMTPPort, SMTPUsername, SMTPPassword        string
	SMTPFrom                                              string
	SMTPTLS                                               bool
	AdminBootstrapEmails                                  []string
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
	return Config{
		MaxBotToken: os.Getenv("MAX_BOT_TOKEN"), MaxAppURL: os.Getenv("MAX_APP_URL"), MaxBotUsername: os.Getenv("MAX_BOT_USERNAME"), MaxCACertFile: os.Getenv("MAX_CA_CERT_FILE"),
		DatabaseURL: cenv("DATABASE_URL"), Port: port, MockStatusEnabled: os.Getenv("MOCK_STATUS_ENABLED") != "false",
		GISHousingBaseURL: baseURL, GISHousingTimeout: timeout, GISHouseCacheTTL: ttl,
		SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: cenvDefault("SMTP_PORT", "587"), SMTPUsername: os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM"), SMTPTLS: os.Getenv("SMTP_TLS") == "true",
		AdminBootstrapEmails: splitCSV(os.Getenv("ADMIN_BOOTSTRAP_EMAILS")),
	}, nil
}

func cenv(name string) string { return os.Getenv(name) }

func cenvDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	var values []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	return values
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
