package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.ServerAddr)
}

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("PKB_SERVER_ADDR", ":9090")
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.ServerAddr)
	assert.Equal(t, "test-client-id", cfg.GoogleClientID)
	assert.Equal(t, "test-secret", cfg.GoogleClientSecret)
}
