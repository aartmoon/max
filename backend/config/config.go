package config

import "os"

type Config struct {
	DatabaseURL, Port                                     string
	MockStatusEnabled                                     bool
	MaxBotToken, MaxAppURL, MaxBotUsername, MaxCACertFile string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{MaxBotToken: os.Getenv("MAX_BOT_TOKEN"), MaxAppURL: os.Getenv("MAX_APP_URL"), MaxBotUsername: os.Getenv("MAX_BOT_USERNAME"), MaxCACertFile: os.Getenv("MAX_CA_CERT_FILE"), DatabaseURL: os.Getenv("DATABASE_URL"), Port: port, MockStatusEnabled: os.Getenv("MOCK_STATUS_ENABLED") != "false"}
}
