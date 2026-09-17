package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	PostgresHost     string
	PostgresPort     int
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	DataDir          string
	BatchSize        int
	LogEvery         int
	AllowDemoData    bool
}

func Load(getenv func(string) string) (Config, error) {
	var cfg Config
	required := []struct {
		name string
		dest *string
	}{
		{"POSTGRES_HOST", &cfg.PostgresHost},
		{"POSTGRES_DB", &cfg.PostgresDB},
		{"POSTGRES_USER", &cfg.PostgresUser},
		{"POSTGRES_PASSWORD", &cfg.PostgresPassword},
	}
	for _, item := range required {
		*item.dest = strings.TrimSpace(getenv(item.name))
		if *item.dest == "" {
			return Config{}, fmt.Errorf("%s is required", item.name)
		}
	}
	var err error
	cfg.PostgresPort, err = positiveInt(getenv("POSTGRES_PORT"), 5432, "POSTGRES_PORT")
	if err != nil {
		return Config{}, err
	}
	cfg.BatchSize, err = positiveInt(getenv("GAR_BATCH_SIZE"), 10000, "GAR_BATCH_SIZE")
	if err != nil {
		return Config{}, err
	}
	cfg.LogEvery, err = positiveInt(getenv("GAR_LOG_EVERY"), 100000, "GAR_LOG_EVERY")
	if err != nil {
		return Config{}, err
	}
	cfg.DataDir = strings.TrimSpace(getenv("GAR_DATA_DIR"))
	if cfg.DataDir == "" {
		cfg.DataDir = "/data/gar"
	}
	if raw := strings.TrimSpace(getenv("GAR_ALLOW_DEMO_DATA")); raw != "" {
		cfg.AllowDemoData, err = strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("GAR_ALLOW_DEMO_DATA: %w", err)
		}
	}
	return cfg, nil
}

func (c Config) DatabaseURL() string {
	u := &url.URL{
		Scheme:  "postgres",
		User:    url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:    fmt.Sprintf("%s:%d", c.PostgresHost, c.PostgresPort),
		Path:    "/" + c.PostgresDB,
		RawPath: "/" + url.PathEscape(c.PostgresDB),
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func positiveInt(raw string, fallback int, name string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
