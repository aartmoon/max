package config

import "os"

type Config struct {
	DatabaseURL, Port string
	MockStatusEnabled bool
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{DatabaseURL: os.Getenv("DATABASE_URL"), Port: port, MockStatusEnabled: os.Getenv("MOCK_STATUS_ENABLED") != "false"}
}
