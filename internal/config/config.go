package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ServerAddr        string
	GoogleClientID    string
	GoogleClientSecret string
	TokenPath         string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerAddr:         envOr("PKB_SERVER_ADDR", ":8080"),
		GoogleClientID:     os.Getenv("PKB_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("PKB_GOOGLE_CLIENT_SECRET"),
		TokenPath:          envOr("PKB_TOKEN_PATH", defaultTokenPath()),
	}
	return cfg, nil
}

// userHomeDir returns the user's home directory. Overridden in tests.
var userHomeDir = os.UserHomeDir

// defaultTokenPath returns the XDG-compliant default path for the OAuth token.
// Uses $XDG_CONFIG_HOME/pkb/token.json if set, otherwise ~/.config/pkb/token.json.
func defaultTokenPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "pkb", "token.json")
	}
	home, err := userHomeDir()
	if err != nil {
		return "token.json"
	}
	return filepath.Join(home, ".config", "pkb", "token.json")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
