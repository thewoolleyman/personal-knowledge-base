package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// Client calls the PKB HTTP API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a Client targeting the given base URL.
func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

// Search queries the /search endpoint and returns results.
// If sources is non-nil, only those connectors are queried.
func (c *Client) Search(ctx context.Context, query string, sources []string) ([]connectors.Result, error) {
	u, err := url.Parse(c.baseURL + "/search")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	params := u.Query()
	params.Set("q", query)
	if len(sources) > 0 {
		params.Set("sources", strings.Join(sources, ","))
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("server returned %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errResp.Error)
	}

	var results []connectors.Result
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return results, nil
}
