package config

import "os"

type Config struct {
	ServerAddr        string
	GoogleClientID    string
	GoogleClientSecret string
	TokenPath         string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerAddr: envOr("PKB_SERVER_ADDR", ":8080"),
		GoogleClientID:    os.Getenv("PKB_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("PKB_GOOGLE_CLIENT_SECRET"),
		TokenPath:         envOr("PKB_TOKEN_PATH", "token.json"),
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
