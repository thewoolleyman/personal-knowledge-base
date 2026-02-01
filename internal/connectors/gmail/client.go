package gmail

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	gm "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// APIClient implements GmailClient using the real Gmail API.
type APIClient struct {
	service *gm.Service
}

// createGmailService creates a Gmail API service. Overridden in tests.
var createGmailService = func(ctx context.Context, opts ...option.ClientOption) (*gm.Service, error) {
	return gm.NewService(ctx, opts...)
}

// NewAPIClient creates a real Gmail API client using the given OAuth2 token source.
func NewAPIClient(ctx context.Context, tokenSource oauth2.TokenSource) (*APIClient, error) {
	srv, err := createGmailService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}
	return &APIClient{service: srv}, nil
}

func (c *APIClient) SearchMessages(ctx context.Context, query string) ([]Message, error) {
	call := c.service.Users.Messages.List("me").
		Q(query).
		MaxResults(20).
		Context(ctx)

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("gmail messages.list: %w", err)
	}

	messages := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		msg, err := c.service.Users.Messages.Get("me", m.Id).
			Format("metadata").
			MetadataHeaders("Subject", "From").
			Context(ctx).
			Do()
		if err != nil {
			continue // skip individual message errors
		}

		var subject, from string
		for _, h := range msg.Payload.Headers {
			switch h.Name {
			case "Subject":
				subject = h.Value
			case "From":
				from = h.Value
			}
		}

		messages = append(messages, Message{
			ID:      m.Id,
			Subject: subject,
			Snippet: msg.Snippet,
			From:    from,
		})
	}

	return messages, nil
}
