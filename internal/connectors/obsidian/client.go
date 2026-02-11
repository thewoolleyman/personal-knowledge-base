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

// buildFolderQuery constructs a Drive API query that searches full text within a folder subtree.
func buildFolderQuery(query, folderID string) string {
	escaped := strings.ReplaceAll(query, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	escapedFolder := strings.ReplaceAll(folderID, `'`, `\'`)
	return fmt.Sprintf("fullText contains '%s' and '%s' in parents and trashed = false", escaped, escapedFolder)
}

// SearchFiles searches for files matching query within the scoped folder.
func (c *FolderScopedClient) SearchFiles(ctx context.Context, query string) ([]gdrive.DriveFile, error) {
	q := buildFolderQuery(query, c.folderID)
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
