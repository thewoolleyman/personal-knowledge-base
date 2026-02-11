package obsidian

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestBuildFolderQuery_SingleFolder(t *testing.T) {
	got := buildFolderQuery("meeting notes", []string{"folder123"})
	assert.Equal(t, "fullText contains 'meeting notes' and 'folder123' in parents and trashed = false", got)
}

func TestBuildFolderQuery_MultipleFolders(t *testing.T) {
	got := buildFolderQuery("meeting notes", []string{"root", "sub1", "sub2"})
	assert.Equal(t, "fullText contains 'meeting notes' and ('root' in parents or 'sub1' in parents or 'sub2' in parents) and trashed = false", got)
}

func TestBuildFolderQuery_EscapesSingleQuotes(t *testing.T) {
	got := buildFolderQuery("it's a test", []string{"folder123"})
	assert.Equal(t, "fullText contains 'it\\'s a test' and 'folder123' in parents and trashed = false", got)
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
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			// Subfolder listing — return empty (no subfolders)
			fmt.Fprint(w, `{"files":[]}`)
			return
		}
		assert.Contains(t, q, "in parents")
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

func TestSearchFiles_QueriesSubfolders(t *testing.T) {
	// Simulate a folder tree: root -> subfolder -> file
	// The search must find files in subfolders, not just direct children.
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			// Subfolder listing request — return one subfolder on first call, empty on second
			if strings.Contains(q, "'rootFolder' in parents") {
				fmt.Fprint(w, `{"files":[{"id":"subfolder1","name":"Projects","mimeType":"application/vnd.google-apps.folder"}]}`)
			} else {
				fmt.Fprint(w, `{"files":[]}`)
			}
			return
		}

		// fullText search — verify it includes both root AND subfolder in parents
		assert.Contains(t, q, "in parents", "Query should search in parents")
		// The query must include both rootFolder and subfolder1
		assert.Contains(t, q, "rootFolder", "Query should include root folder")
		assert.Contains(t, q, "subfolder1", "Query should include discovered subfolder")
		fmt.Fprint(w, `{"files":[{"id":"file1","name":"Grocery List.md","mimeType":"text/markdown","webViewLink":"https://drive.google.com/file1","description":""}]}`)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "rootFolder")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	files, err := client.SearchFiles(context.Background(), "Hummus")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "Grocery List.md", files[0].Name)
}

func TestSearchFiles_APIError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			// Subfolder listing succeeds
			fmt.Fprint(w, `{"files":[]}`)
			return
		}
		// Search fails
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

func TestSearchFiles_SubfolderListingError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// All requests fail — subfolder listing will fail first
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "folder123")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchFiles(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collect subfolder IDs")
}
