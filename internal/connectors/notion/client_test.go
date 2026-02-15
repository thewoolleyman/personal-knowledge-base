package notion

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIClient_SetsToken(t *testing.T) {
	client := NewAPIClient("ntn_test_token")
	assert.NotNil(t, client)
}

func TestAPIClient_Search_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "Bearer ntn_test", r.Header.Get("Authorization"))
		assert.Equal(t, "2022-06-28", r.Header.Get("Notion-Version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "page-1",
					"object": "page",
					"url": "https://www.notion.so/Page-One-abc123",
					"properties": {
						"title": {
							"type": "title",
							"title": [{"plain_text": "Test Page One"}]
						},
						"Name": {
							"type": "title",
							"title": [{"plain_text": "Test Page One"}]
						}
					}
				},
				{
					"id": "page-2",
					"object": "page",
					"url": "https://www.notion.so/Page-Two-def456",
					"properties": {
						"title": {
							"type": "title",
							"title": [{"plain_text": "Test Page Two"}]
						}
					}
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "test page")
	require.NoError(t, err)
	require.Len(t, pages, 2)

	assert.Equal(t, "page-1", pages[0].ID)
	assert.Equal(t, "Test Page One", pages[0].Title)
	assert.Equal(t, "https://www.notion.so/Page-One-abc123", pages[0].URL)

	assert.Equal(t, "page-2", pages[1].ID)
	assert.Equal(t, "Test Page Two", pages[1].Title)
	assert.Equal(t, "https://www.notion.so/Page-Two-def456", pages[1].URL)
}

func TestAPIClient_Search_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results": []}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "nothing")
	require.NoError(t, err)
	assert.Empty(t, pages)
}

func TestAPIClient_Search_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"object":"error","status":401,"message":"API token is invalid."}`)
	}))
	defer srv.Close()

	client := NewAPIClient("bad_token")
	client.baseURL = srv.URL

	_, err := client.Search(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notion search")
}

func TestAPIClient_Search_NetworkError(t *testing.T) {
	client := NewAPIClient("ntn_test")
	client.baseURL = "http://127.0.0.1:1" // nothing listening

	_, err := client.Search(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notion search request")
}

func TestAPIClient_Search_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	_, err := client.Search(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notion search decode")
}

func TestAPIClient_Search_DatabaseResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "db-1",
					"object": "database",
					"url": "https://www.notion.so/db-1",
					"title": [{"plain_text": "My Database"}]
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "database")
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "My Database", pages[0].Title)
	assert.Equal(t, "https://www.notion.so/db-1", pages[0].URL)
}

func TestAPIClient_Search_PageWithNoTitle_FilteredOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "page-no-title",
					"object": "page",
					"url": "https://www.notion.so/Untitled-xyz",
					"properties": {}
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "test")
	require.NoError(t, err)
	assert.Empty(t, pages, "Pages with no title should be filtered out")
}

func TestAPIClient_Search_DatabaseWithEmptyTitle_FilteredOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "db-empty",
					"object": "database",
					"url": "https://www.notion.so/db-empty",
					"title": [{"plain_text": ""}]
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "test")
	require.NoError(t, err)
	assert.Empty(t, pages, "Database with empty title should be filtered out")
}

func TestAPIClient_Search_FiltersNonMatchingResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "page-match",
					"object": "page",
					"url": "https://www.notion.so/Match-abc",
					"properties": {
						"title": {
							"type": "title",
							"title": [{"plain_text": "My Test Page"}]
						}
					}
				},
				{
					"id": "page-nomatch",
					"object": "page",
					"url": "https://www.notion.so/NoMatch-def",
					"properties": {
						"title": {
							"type": "title",
							"title": [{"plain_text": "Unrelated Document"}]
						}
					}
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "Test Page")
	require.NoError(t, err)
	require.Len(t, pages, 1, "Should filter out non-matching results")
	assert.Equal(t, "My Test Page", pages[0].Title)
}

func TestAPIClient_Search_FilterIsCaseInsensitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"results": [
				{
					"id": "page-1",
					"object": "page",
					"url": "https://www.notion.so/Page-1",
					"properties": {
						"title": {
							"type": "title",
							"title": [{"plain_text": "UPPERCASE TITLE"}]
						}
					}
				}
			]
		}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	pages, err := client.Search(context.Background(), "uppercase title")
	require.NoError(t, err)
	require.Len(t, pages, 1, "Case-insensitive match should find the page")
}

func TestAPIClient_Search_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results": []}`)
	}))
	defer srv.Close()

	client := NewAPIClient("ntn_test")
	client.baseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Search(ctx, "test")
	require.Error(t, err)
}
