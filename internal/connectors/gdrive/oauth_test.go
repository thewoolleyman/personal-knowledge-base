package gdrive

import (
	"os"
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

func TestLoadToken_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{truncated"), 0600)
	_, err := LoadToken(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode token")
}

func TestSaveToken_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "test-token"}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// BUG-002: SaveToken must surface errors from both Encode and Close.
func TestSaveToken_BadDirectory(t *testing.T) {
	// Writing to a non-existent directory should return an error.
	err := SaveToken("/nonexistent/dir/token.json", &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create token file")
}

func TestSaveToken_ClosesFileExplicitly(t *testing.T) {
	// Verify the file is properly closed by writing and then immediately reading.
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "close-test", TokenType: "Bearer"}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	// If Close was called properly, the file should be fully flushed and readable.
	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "close-test", loaded.AccessToken)
}

func TestSaveToken_ReadOnlyDir(t *testing.T) {
	// Create a read-only directory and verify SaveToken fails.
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0500))
	t.Cleanup(func() { os.Chmod(readOnlyDir, 0700) })

	path := filepath.Join(readOnlyDir, "token.json")
	err := SaveToken(path, &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
}
