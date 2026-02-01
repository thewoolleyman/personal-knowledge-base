package gdrive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSearchQuery_EscapesSingleQuotes(t *testing.T) {
	got := buildSearchQuery("it's a test")
	assert.Equal(t, "fullText contains 'it\\'s a test' and trashed = false", got)
}

func TestBuildSearchQuery_NoSpecialChars(t *testing.T) {
	got := buildSearchQuery("simple query")
	assert.Equal(t, "fullText contains 'simple query' and trashed = false", got)
}

func TestBuildSearchQuery_MultipleQuotes(t *testing.T) {
	got := buildSearchQuery("it's Bob's file")
	assert.Equal(t, "fullText contains 'it\\'s Bob\\'s file' and trashed = false", got)
}

func TestBuildSearchQuery_BackslashBeforeQuote(t *testing.T) {
	got := buildSearchQuery("test\\'already")
	assert.Equal(t, "fullText contains 'test\\\\\\'already' and trashed = false", got)
}
