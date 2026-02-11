package obsidian

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

func TestBuildFolderQuery_Basic(t *testing.T) {
	got := buildFolderQuery("meeting notes", "folder123")
	assert.Equal(t, "fullText contains 'meeting notes' and 'folder123' in parents and trashed = false", got)
}

func TestBuildFolderQuery_EscapesSingleQuotes(t *testing.T) {
	got := buildFolderQuery("it's a test", "folder123")
	assert.Equal(t, "fullText contains 'it\\'s a test' and 'folder123' in parents and trashed = false", got)
}

func TestBuildFolderQuery_NoSpecialChars(t *testing.T) {
	got := buildFolderQuery("simple query", "abc")
	assert.Equal(t, "fullText contains 'simple query' and 'abc' in parents and trashed = false", got)
}

func TestNewFolderScopedClient_Success(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "folder123")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "folder123", client.folderID)
}

func TestNewFolderScopedClient_ServiceError(t *testing.T) {
	orig := createDriveService
	createDriveService = func(_ context.Context, _ ...option.ClientOption) (*drive.Service, error) {
		return nil, fmt.Errorf("service creation failed")
	}
	t.Cleanup(func() { createDriveService = orig })

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	_, err := NewFolderScopedClient(context.Background(), ts, "folder123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create drive service")
}

func TestSearchFiles_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the query includes folder scoping
		q := r.URL.Query().Get("q")
		assert.Contains(t, q, "in parents")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"files":[{"id":"1","name":"note.md","mimeType":"text/markdown","webViewLink":"https://drive.google.com/1","description":"A vault note"}]}`)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "folder123")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	files, err := client.SearchFiles(context.Background(), "test")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "1", files[0].ID)
	assert.Equal(t, "note.md", files[0].Name)
	assert.Equal(t, "text/markdown", files[0].MimeType)
	assert.Equal(t, "A vault note", files[0].Description)
}

func TestSearchFiles_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "folder123")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchFiles(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "drive files.list (obsidian)")
}
