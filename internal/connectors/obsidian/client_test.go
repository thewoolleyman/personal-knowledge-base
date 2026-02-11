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

// driveHandler creates an httptest handler that responds to folder listing and search queries.
// folders maps folderID -> list of parentIDs (for the folder listing call).
// searchResponse is returned for fullText search queries.
func driveHandler(t *testing.T, folders map[string][]string, searchResponse string, searchAssertions func(q string)) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			// Build folder listing response from the map.
			var parts []string
			for id, parents := range folders {
				var parentJSON []string
				for _, p := range parents {
					parentJSON = append(parentJSON, fmt.Sprintf("%q", p))
				}
				parts = append(parts, fmt.Sprintf(`{"id":%q,"parents":[%s]}`, id, strings.Join(parentJSON, ",")))
			}
			fmt.Fprintf(w, `{"files":[%s]}`, strings.Join(parts, ","))
			return
		}

		if searchAssertions != nil {
			searchAssertions(q)
		}
		fmt.Fprint(w, searchResponse)
	})
}

func TestSearchFiles_Success(t *testing.T) {
	// No subfolders — just the root folder.
	handler := driveHandler(t, map[string][]string{}, `{"files":[{"id":"1","name":"note.md","mimeType":"text/markdown","webViewLink":"https://drive.google.com/1","description":"A vault note"}]}`, func(q string) {
		assert.Contains(t, q, "in parents")
	})
	srv := httptest.NewServer(handler)
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
	// Folder tree: rootFolder -> subfolder1 (Projects)
	// The single folder-listing call returns subfolder1 with rootFolder as parent.
	folders := map[string][]string{
		"subfolder1": {"rootFolder"},
	}
	handler := driveHandler(t, folders,
		`{"files":[{"id":"file1","name":"Grocery List.md","mimeType":"text/markdown","webViewLink":"https://drive.google.com/file1","description":""}]}`,
		func(q string) {
			assert.Contains(t, q, "rootFolder", "Query should include root folder")
			assert.Contains(t, q, "subfolder1", "Query should include discovered subfolder")
		},
	)
	srv := httptest.NewServer(handler)
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

func TestSearchFiles_DeepNesting(t *testing.T) {
	// rootFolder -> sub1 -> sub2 (3 levels deep, still only 1 API call for folders)
	folders := map[string][]string{
		"sub1": {"rootFolder"},
		"sub2": {"sub1"},
	}
	handler := driveHandler(t, folders,
		`{"files":[{"id":"f1","name":"deep.md","mimeType":"text/markdown","webViewLink":"https://example.com","description":""}]}`,
		func(q string) {
			assert.Contains(t, q, "rootFolder")
			assert.Contains(t, q, "sub1")
			assert.Contains(t, q, "sub2")
		},
	)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "rootFolder")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	files, err := client.SearchFiles(context.Background(), "test")
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestSearchFiles_ExcludesUnrelatedFolders(t *testing.T) {
	// unrelated folder exists in Drive but is NOT under rootFolder
	folders := map[string][]string{
		"subfolder1": {"rootFolder"},
		"unrelated":  {"otherRoot"},
	}
	handler := driveHandler(t, folders,
		`{"files":[]}`,
		func(q string) {
			assert.Contains(t, q, "rootFolder")
			assert.Contains(t, q, "subfolder1")
			assert.NotContains(t, q, "unrelated", "Should not include folders outside the target subtree")
			assert.NotContains(t, q, "otherRoot")
		},
	)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "rootFolder")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchFiles(context.Background(), "test")
	require.NoError(t, err)
}

func TestSearchFiles_FolderListingPagination(t *testing.T) {
	// Folder listing returns results across two pages.
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			pt := r.URL.Query().Get("pageToken")
			if pt == "" && page == 0 {
				page++
				fmt.Fprint(w, `{"files":[{"id":"sub1","parents":["rootFolder"]}],"nextPageToken":"page2"}`)
			} else {
				fmt.Fprint(w, `{"files":[{"id":"sub2","parents":["sub1"]}]}`)
			}
			return
		}
		// Search — verify all folders included
		assert.Contains(t, q, "rootFolder")
		assert.Contains(t, q, "sub1")
		assert.Contains(t, q, "sub2")
		fmt.Fprint(w, `{"files":[]}`)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewFolderScopedClient(context.Background(), ts, "rootFolder")
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchFiles(context.Background(), "test")
	require.NoError(t, err)
}

func TestSearchFiles_APIError(t *testing.T) {
	// Folder listing succeeds, search fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(q, "mimeType = 'application/vnd.google-apps.folder'") {
			fmt.Fprint(w, `{"files":[]}`)
			return
		}
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

func TestSearchFiles_FolderListingError(t *testing.T) {
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
	assert.Contains(t, err.Error(), "collect subfolder IDs")
}
