//go:build integration

package gdrive

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}

func setupIntegrationClient(t *testing.T) *APIClient {
	t.Helper()
	clientID := requireEnv(t, "PKB_GOOGLE_CLIENT_ID")
	clientSecret := requireEnv(t, "PKB_GOOGLE_CLIENT_SECRET")
	tokenPath := os.Getenv("PKB_TOKEN_PATH")
	if tokenPath == "" {
		tokenPath = "token.json"
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{drive.DriveReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	tok, err := LoadToken(tokenPath)
	require.NoError(t, err, "Failed to load token from %s — run OAuth flow first", tokenPath)

	ctx := context.Background()
	client, err := NewAPIClient(ctx, cfg.TokenSource(ctx, tok))
	require.NoError(t, err)
	return client
}

func TestIntegration_GDrive_SearchReturnsResults(t *testing.T) {
	client := setupIntegrationClient(t)

	// Search for something likely to exist in any Google Drive
	// The Obsidian vault mirror should have markdown files
	results, err := client.SearchFiles(context.Background(), "md")

	require.NoError(t, err)
	assert.NotEmpty(t, results, "Expected at least one result from Google Drive")

	// Verify result fields are populated
	for _, r := range results {
		assert.NotEmpty(t, r.ID, "File ID should not be empty")
		assert.NotEmpty(t, r.Name, "File name should not be empty")
	}
}

func TestIntegration_GDrive_ConnectorEndToEnd(t *testing.T) {
	apiClient := setupIntegrationClient(t)
	connector := NewConnector(apiClient)

	assert.Equal(t, "google-drive", connector.Name())

	results, err := connector.Search(context.Background(), "md")
	require.NoError(t, err)
	assert.NotEmpty(t, results, "Expected search results from connector")

	// Verify connector results have expected fields
	for _, r := range results {
		assert.NotEmpty(t, r.Title)
		assert.Equal(t, "google-drive", r.Source)
	}
}

func TestIntegration_GDrive_EmptyQueryReturnsResults(t *testing.T) {
	client := setupIntegrationClient(t)

	// Even a broad search should return something
	results, err := client.SearchFiles(context.Background(), "the")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}
