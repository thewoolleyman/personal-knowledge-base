package gdrive

import (
	"context"
	"fmt"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// DriveFile represents a file returned from the Google Drive API.
type DriveFile struct {
	ID          string
	Name        string
	MimeType    string
	WebViewLink string
}

// DriveClient abstracts the Google Drive API for testability.
type DriveClient interface {
	SearchFiles(ctx context.Context, query string) ([]DriveFile, error)
}

// Connector implements connectors.Connector for Google Drive.
type Connector struct {
	client DriveClient
}

// NewConnector creates a Google Drive connector with the given client.
func NewConnector(client DriveClient) *Connector {
	return &Connector{client: client}
}

func (c *Connector) Name() string {
	return "google-drive"
}

func (c *Connector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	files, err := c.client.SearchFiles(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("google drive search: %w", err)
	}

	results := make([]connectors.Result, len(files))
	for i, f := range files {
		results[i] = connectors.Result{
			Title:  f.Name,
			URL:    f.WebViewLink,
			Source: "google-drive",
		}
	}

	return results, nil
}
