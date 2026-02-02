package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
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

func TestLoad_TokenPathDefault_UsesXDGConfigHome(t *testing.T) {
	t.Setenv("PKB_TOKEN_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/test-xdg-config")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/test-xdg-config", "pkb", "token.json"), cfg.TokenPath)
}

func TestLoad_TokenPathDefault_FallsBackToHomeConfig(t *testing.T) {
	t.Setenv("PKB_TOKEN_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/test-home")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/test-home", ".config", "pkb", "token.json"), cfg.TokenPath)
}

func TestLoad_TokenPathDefault_FallsBackToTokenJSON_WhenHomeDirFails(t *testing.T) {
	t.Setenv("PKB_TOKEN_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	orig := userHomeDir
	userHomeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	t.Cleanup(func() { userHomeDir = orig })

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "token.json", cfg.TokenPath)
}

func TestLoad_TokenPathEnvOverride(t *testing.T) {
	t.Setenv("PKB_TOKEN_PATH", "/custom/token.json")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/custom/token.json", cfg.TokenPath)
}

func TestLoad_CallsLoadDotenv(t *testing.T) {
	called := false
	orig := loadDotenv
	loadDotenv = func() { called = true }
	t.Cleanup(func() { loadDotenv = orig })

	_, _ = Load()
	assert.True(t, called, "Load() must call loadDotenv()")
}

func TestLoad_DotenvFilePopulatesConfig(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("PKB_GOOGLE_CLIENT_ID=dotenv-id\nPKB_GOOGLE_CLIENT_SECRET=dotenv-secret\n"), 0644))

	orig := loadDotenv
	loadDotenv = func() { _ = godotenv.Load(envPath) }
	t.Cleanup(func() { loadDotenv = orig })

	// Register for cleanup, then unset so godotenv can set them.
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "")
	os.Unsetenv("PKB_GOOGLE_CLIENT_ID")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "")
	os.Unsetenv("PKB_GOOGLE_CLIENT_SECRET")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "dotenv-id", cfg.GoogleClientID)
	assert.Equal(t, "dotenv-secret", cfg.GoogleClientSecret)
}

func TestLoad_EnvVarsTakePrecedenceOverDotenv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("PKB_GOOGLE_CLIENT_ID=dotenv-id\n"), 0644))

	orig := loadDotenv
	loadDotenv = func() { _ = godotenv.Load(envPath) }
	t.Cleanup(func() { loadDotenv = orig })

	t.Setenv("PKB_GOOGLE_CLIENT_ID", "real-id")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "real-id", cfg.GoogleClientID)
}
