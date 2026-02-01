package gdrive

import (
	"context"
	"fmt"
	"strings"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"golang.org/x/oauth2"
)

// APIClient implements DriveClient using the real Google Drive API.
type APIClient struct {
	service *drive.Service
}

// createDriveService creates a Drive API service. Overridden in tests.
var createDriveService = func(ctx context.Context, opts ...option.ClientOption) (*drive.Service, error) {
	return drive.NewService(ctx, opts...)
}

// NewAPIClient creates a real Drive API client using the given OAuth2 token source.
func NewAPIClient(ctx context.Context, tokenSource oauth2.TokenSource) (*APIClient, error) {
	srv, err := createDriveService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	return &APIClient{service: srv}, nil
}

// buildSearchQuery constructs a Drive API query string, escaping single quotes
// in user input to prevent query injection.
func buildSearchQuery(query string) string {
	escaped := strings.ReplaceAll(query, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return fmt.Sprintf("fullText contains '%s' and trashed = false", escaped)
}

func (c *APIClient) SearchFiles(ctx context.Context, query string) ([]DriveFile, error) {
	q := buildSearchQuery(query)
	call := c.service.Files.List().
		Q(q).
		Fields("files(id, name, mimeType, webViewLink)").
		PageSize(50).
		Context(ctx)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("drive files.list: %w", err)
	}

	files := make([]DriveFile, len(resp.Files))
	for i, f := range resp.Files {
		files[i] = DriveFile{
			ID:          f.Id,
			Name:        f.Name,
			MimeType:    f.MimeType,
			WebViewLink: f.WebViewLink,
		}
	}

	return files, nil
}
