package notion

import (
	"context"
	"fmt"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// Connector implements connectors.Connector for Notion.
type Connector struct {
	client NotionClient
}

// NewConnector creates a Notion connector with the given client.
func NewConnector(client NotionClient) *Connector {
	return &Connector{client: client}
}

func (c *Connector) Name() string {
	return "notion"
}

func (c *Connector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	pages, err := c.client.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("notion search: %w", err)
	}

	results := make([]connectors.Result, len(pages))
	for i, p := range pages {
		results[i] = connectors.Result{
			Title:  p.Title,
			URL:    p.URL,
			Source: "notion",
		}
	}

	return results, nil
}
