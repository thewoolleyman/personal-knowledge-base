package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://api.notion.com"

// Page represents a Notion page or database returned from the API.
type Page struct {
	ID    string
	Title string
	URL   string
}

// NotionClient abstracts the Notion API for testability.
type NotionClient interface {
	Search(ctx context.Context, query string) ([]Page, error)
}

// APIClient implements NotionClient using the Notion REST API.
type APIClient struct {
	token   string
	baseURL string
	http    *http.Client
}

// NewAPIClient creates a Notion API client with the given integration token.
func NewAPIClient(token string) *APIClient {
	return &APIClient{
		token:   token,
		baseURL: defaultBaseURL,
		http:    http.DefaultClient,
	}
}

// searchRequest is the JSON body for POST /v1/search.
type searchRequest struct {
	Query string `json:"query"`
}

// searchResponse is the JSON response from POST /v1/search.
type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	ID         string                       `json:"id"`
	Object     string                       `json:"object"`
	URL        string                       `json:"url"`
	Properties map[string]searchProperty    `json:"properties"`
	Title      []searchRichText             `json:"title"` // for database objects
}

type searchProperty struct {
	Type  string           `json:"type"`
	Title []searchRichText `json:"title"`
}

type searchRichText struct {
	PlainText string `json:"plain_text"`
}

func (c *APIClient) Search(ctx context.Context, query string) ([]Page, error) {
	body, _ := json.Marshal(searchRequest{Query: query}) // simple struct, cannot fail

	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/search", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", "2022-06-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("notion search: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("notion search decode: %w", err)
	}

	queryLower := strings.ToLower(query)
	pages := make([]Page, 0, len(result.Results))
	for _, r := range result.Results {
		title := extractTitle(r)
		if !strings.Contains(strings.ToLower(title), queryLower) {
			continue
		}
		pages = append(pages, Page{
			ID:    r.ID,
			Title: title,
			URL:   r.URL,
		})
	}

	return pages, nil
}

// extractTitle gets the title from a page or database result.
func extractTitle(r searchResult) string {
	// Database objects have title at top level
	if r.Object == "database" {
		for _, t := range r.Title {
			if t.PlainText != "" {
				return t.PlainText
			}
		}
	}

	// Page objects have title in properties
	for _, prop := range r.Properties {
		if prop.Type == "title" {
			for _, t := range prop.Title {
				if t.PlainText != "" {
					return t.PlainText
				}
			}
		}
	}

	return ""
}
