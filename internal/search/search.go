package search

import (
	"context"
	"fmt"
	"sync"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// Engine fans out search queries to multiple connectors concurrently.
type Engine struct {
	connectors []connectors.Connector
}

// New creates a search engine with the given connectors.
func New(cs ...connectors.Connector) *Engine {
	return &Engine{connectors: cs}
}

// Search queries all connectors concurrently and aggregates results.
// If some connectors fail, results from healthy ones are still returned.
// Returns an error only if ALL connectors fail.
func (e *Engine) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	if len(e.connectors) == 0 {
		return nil, nil
	}

	type result struct {
		results []connectors.Result
		err     error
		name    string
	}

	ch := make(chan result, len(e.connectors))
	var wg sync.WaitGroup

	for _, c := range e.connectors {
		wg.Add(1)
		go func(c connectors.Connector) {
			defer wg.Done()
			res, err := c.Search(ctx, query)
			ch <- result{results: res, err: err, name: c.Name()}
		}(c)
	}

	wg.Wait()
	close(ch)

	var all []connectors.Result
	var errs []error

	for r := range ch {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, r.err))
			continue
		}
		all = append(all, r.results...)
	}

	if len(errs) == len(e.connectors) {
		return nil, fmt.Errorf("all connectors failed: %v", errs)
	}

	return all, nil
}
