package gmail

import (
	"context"
	"fmt"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// Message represents an email message returned from the Gmail API.
type Message struct {
	ID      string
	Subject string
	Snippet string
	From    string
}

// GmailClient abstracts the Gmail API for testability.
type GmailClient interface {
	SearchMessages(ctx context.Context, query string) ([]Message, error)
}

// Connector implements connectors.Connector for Gmail.
type Connector struct {
	client GmailClient
}

// NewConnector creates a Gmail connector with the given client.
func NewConnector(client GmailClient) *Connector {
	return &Connector{client: client}
}

func (c *Connector) Name() string {
	return "gmail"
}

func (c *Connector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	messages, err := c.client.SearchMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gmail search: %w", err)
	}

	results := make([]connectors.Result, len(messages))
	for i, m := range messages {
		results[i] = connectors.Result{
			Title:   m.Subject,
			Snippet: m.Snippet,
			URL:     fmt.Sprintf("https://mail.google.com/mail/u/0/#inbox/%s", m.ID),
			Source:  "gmail",
		}
	}

	return results, nil
}
