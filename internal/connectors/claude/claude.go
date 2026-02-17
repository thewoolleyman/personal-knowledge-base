package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// Connector implements connectors.Connector for Claude.ai exported conversations.
type Connector struct {
	client ClaudeClient
}

// NewConnector creates a Claude connector with the given client.
func NewConnector(client ClaudeClient) *Connector {
	return &Connector{client: client}
}

func (c *Connector) Name() string {
	return "claude"
}

func (c *Connector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	conversations, err := c.client.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("claude search: %w", err)
	}

	results := make([]connectors.Result, len(conversations))
	for i, conv := range conversations {
		results[i] = connectors.Result{
			Title:   conv.Name,
			URL:     "https://claude.ai/chat/" + conv.UUID,
			Snippet: firstMatchingSnippet(conv, query),
			Source:  "claude",
		}
	}

	return results, nil
}

// firstMatchingSnippet returns the text of the first message matching the query,
// truncated to 200 characters.
func firstMatchingSnippet(conv Conversation, query string) string {
	queryLower := strings.ToLower(query)
	for _, msg := range conv.ChatMessages {
		if strings.Contains(strings.ToLower(msg.Text), queryLower) {
			return truncate(msg.Text, 200)
		}
	}
	// If match was on conversation name, use the first message as snippet.
	if len(conv.ChatMessages) > 0 {
		return truncate(conv.ChatMessages[0].Text, 200)
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
