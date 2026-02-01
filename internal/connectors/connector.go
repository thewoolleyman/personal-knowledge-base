package connectors

import "context"

// Result represents a single search result from any connector.
type Result struct {
	Title   string
	Snippet string
	URL     string
	Source  string
}

// Connector is the interface that each data source implements.
type Connector interface {
	Search(ctx context.Context, query string) ([]Result, error)
	Name() string
}
