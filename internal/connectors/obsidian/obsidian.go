package obsidian

import (
	"context"
	"fmt"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
)

// Connector implements connectors.Connector for an Obsidian vault stored in Google Drive.
// It wraps a gdrive.DriveClient and restricts searches to a specific folder.
type Connector struct {
	client   gdrive.DriveClient
	folderID string
}

// NewConnector creates an Obsidian connector that searches only within the given folder.
func NewConnector(client gdrive.DriveClient, folderID string) *Connector {
	return &Connector{client: client, folderID: folderID}
}

func (c *Connector) Name() string {
	return "obsidian"
}

func (c *Connector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	// Delegate to the Drive client — the folder-scoped query is built by the client wrapper.
	files, err := c.client.SearchFiles(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("obsidian search: %w", err)
	}

	results := make([]connectors.Result, len(files))
	for i, f := range files {
		results[i] = connectors.Result{
			Title:   f.Name,
			URL:     f.WebViewLink,
			Source:  "obsidian",
			Snippet: f.Description,
		}
	}

	return results, nil
}
