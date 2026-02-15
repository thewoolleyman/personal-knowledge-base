//go:build integration

package notion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// internal/connectors/notion/ is three levels below project root
	if root, err := filepath.Abs("../../.."); err == nil {
		_ = godotenv.Load(filepath.Join(root, ".env"))
	}
	os.Exit(m.Run())
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s not set — add it to your .env file", key)
	}
	return v
}

func setupIntegrationClient(t *testing.T) *APIClient {
	t.Helper()
	token := requireEnv(t, "PKB_NOTION_TOKEN")
	return NewAPIClient(token)
}

const testPageTitle = "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE (Notion)"

func TestIntegration_Notion_SearchReturnsOnlyMatchingPage(t *testing.T) {
	client := setupIntegrationClient(t)

	results, err := client.Search(context.Background(), "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE")

	require.NoError(t, err)
	require.Len(t, results, 1, "Expected exactly one result matching the test page")
	assert.Equal(t, testPageTitle, results[0].Title)
	assert.NotEmpty(t, results[0].ID, "Page ID should not be empty")
	assert.NotEmpty(t, results[0].URL, "Page URL should not be empty")
}

func TestIntegration_Notion_ConnectorEndToEnd(t *testing.T) {
	apiClient := setupIntegrationClient(t)
	connector := NewConnector(apiClient)

	assert.Equal(t, "notion", connector.Name())

	results, err := connector.Search(context.Background(), "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE")
	require.NoError(t, err)
	require.Len(t, results, 1, "Expected exactly one matching result from connector")
	assert.Equal(t, testPageTitle, results[0].Title)
	assert.Equal(t, "notion", results[0].Source)
}

func TestIntegration_Notion_NonMatchingQueryReturnsEmpty(t *testing.T) {
	client := setupIntegrationClient(t)

	results, err := client.Search(context.Background(), "zzz_nonexistent_query_that_matches_nothing_xyz")
	require.NoError(t, err)
	assert.Empty(t, results, "Query matching nothing should return empty results")
}
