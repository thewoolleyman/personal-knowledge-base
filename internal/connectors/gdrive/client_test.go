package gdrive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestBuildSearchQuery_EscapesSingleQuotes(t *testing.T) {
	got := buildSearchQuery("it's a test")
	assert.Equal(t, "fullText contains 'it\\'s a test' and trashed = false", got)
}

func TestBuildSearchQuery_NoSpecialChars(t *testing.T) {
	got := buildSearchQuery("simple query")
	assert.Equal(t, "fullText contains 'simple query' and trashed = false", got)
}

func TestBuildSearchQuery_MultipleQuotes(t *testing.T) {
	got := buildSearchQuery("it's Bob's file")
	assert.Equal(t, "fullText contains 'it\\'s Bob\\'s file' and trashed = false", got)
}

func TestBuildSearchQuery_BackslashBeforeQuote(t *testing.T) {
	got := buildSearchQuery("test\\'already")
	assert.Equal(t, "fullText contains 'test\\\\\\'already' and trashed = false", got)
}

func TestNewAPIClient_Success(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.service)
}

func TestNewAPIClient_ServiceError(t *testing.T) {
	orig := createDriveService
	createDriveService = func(_ context.Context, _ ...option.ClientOption) (*drive.Service, error) {
		return nil, fmt.Errorf("service creation failed")
	}
	t.Cleanup(func() { createDriveService = orig })

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	_, err := NewAPIClient(context.Background(), ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create drive service")
}

func TestSearchFiles_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"files":[{"id":"1","name":"test.txt","mimeType":"text/plain","webViewLink":"https://drive.google.com/1","description":"A test document"}]}`)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	files, err := client.SearchFiles(context.Background(), "test")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "1", files[0].ID)
	assert.Equal(t, "test.txt", files[0].Name)
	assert.Equal(t, "text/plain", files[0].MimeType)
	assert.Equal(t, "https://drive.google.com/1", files[0].WebViewLink)
	assert.Equal(t, "A test document", files[0].Description)
}

func TestSearchFiles_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchFiles(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "drive files.list")
}
