package gdrive

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", loaded.AccessToken)
	assert.Equal(t, "test-refresh-token", loaded.RefreshToken)
	assert.Equal(t, "Bearer", loaded.TokenType)
}

func TestLoadToken_FileNotFound(t *testing.T) {
	_, err := LoadToken("/nonexistent/token.json")
	assert.Error(t, err)
}
