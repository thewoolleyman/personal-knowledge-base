package obsidian

import (
	"context"
	"fmt"
	"strings"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"golang.org/x/oauth2"
)

// FolderScopedClient wraps a Drive API service and restricts searches to a folder subtree.
type FolderScopedClient struct {
	service  *drive.Service
	folderID string
}

// createDriveService creates a Drive API service. Overridden in tests.
var createDriveService = func(ctx context.Context, opts ...option.ClientOption) (*drive.Service, error) {
	return drive.NewService(ctx, opts...)
}

// NewFolderScopedClient creates a Drive client that searches only within folderID.
func NewFolderScopedClient(ctx context.Context, tokenSource oauth2.TokenSource, folderID string) (*FolderScopedClient, error) {
	srv, err := createDriveService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	return &FolderScopedClient{service: srv, folderID: folderID}, nil
}

// escapeQuery escapes a string for use in a Drive API query.
func escapeQuery(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// buildFolderQuery constructs a Drive API query that searches full text within
// multiple folders (the root and all its subfolders).
func buildFolderQuery(query string, folderIDs []string) string {
	escaped := escapeQuery(query)
	if len(folderIDs) == 1 {
		return fmt.Sprintf("fullText contains '%s' and '%s' in parents and trashed = false", escaped, escapeQuery(folderIDs[0]))
	}
	var parts []string
	for _, id := range folderIDs {
		parts = append(parts, fmt.Sprintf("'%s' in parents", escapeQuery(id)))
	}
	return fmt.Sprintf("fullText contains '%s' and (%s) and trashed = false", escaped, strings.Join(parts, " or "))
}

// collectSubfolderIDs finds all subfolder IDs under parentID using a single
// API call to list all folders in Drive, then filters client-side.
// This is O(1) API calls regardless of folder tree depth.
func (c *FolderScopedClient) collectSubfolderIDs(ctx context.Context, parentID string) ([]string, error) {
	// One API call: get all folders with their parent IDs.
	var allFolders []*drive.File
	pageToken := ""
	for {
		call := c.service.Files.List().
			Q("mimeType = 'application/vnd.google-apps.folder' and trashed = false").
			Fields("nextPageToken, files(id, parents)").
			PageSize(1000).
			IncludeItemsFromAllDrives(true).
			SupportsAllDrives(true).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list all folders: %w", err)
		}
		allFolders = append(allFolders, resp.Files...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Build parent -> children map.
	children := make(map[string][]string)
	for _, f := range allFolders {
		for _, p := range f.Parents {
			children[p] = append(children[p], f.Id)
		}
	}

	// BFS from parentID to find all descendants.
	result := []string{parentID}
	queue := []string{parentID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			result = append(result, child)
			queue = append(queue, child)
		}
	}
	return result, nil
}

// SearchFiles searches for files matching query within the scoped folder and all subfolders.
func (c *FolderScopedClient) SearchFiles(ctx context.Context, query string) ([]gdrive.DriveFile, error) {
	folderIDs, err := c.collectSubfolderIDs(ctx, c.folderID)
	if err != nil {
		return nil, fmt.Errorf("collect subfolder IDs: %w", err)
	}

	q := buildFolderQuery(query, folderIDs)
	call := c.service.Files.List().
		Q(q).
		Fields("files(id, name, mimeType, webViewLink, description)").
		PageSize(50).
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Context(ctx)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("drive files.list (obsidian): %w", err)
	}

	files := make([]gdrive.DriveFile, len(resp.Files))
	for i, f := range resp.Files {
		files[i] = gdrive.DriveFile{
			ID:          f.Id,
			Name:        f.Name,
			MimeType:    f.MimeType,
			WebViewLink: f.WebViewLink,
			Description: f.Description,
		}
	}

	return files, nil
}
