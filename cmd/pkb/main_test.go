package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_ReturnsNilOnSuccess(t *testing.T) {
	err := run([]string{}, nil)
	assert.NoError(t, err)
}

func TestSearchCommand_PrintsResults(t *testing.T) {
	mockSearch := func(_ context.Context, query string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "Test Doc", URL: "https://example.com/doc", Source: "mock"},
			{Title: "Another Doc", URL: "https://example.com/doc2", Source: "mock"},
		}, nil
	}

	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test query"}, mockSearch, &buf)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Test Doc")
	assert.Contains(t, output, "https://example.com/doc")
	assert.Contains(t, output, "Another Doc")
}

func TestSearchCommand_NoQuery(t *testing.T) {
	err := run([]string{"search"}, nil)
	assert.Error(t, err)
}
