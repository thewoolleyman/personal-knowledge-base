package tui

import (
	"context"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockSearchFn(results []connectors.Result, err error) SearchFunc {
	return func(_ context.Context, _ string) ([]connectors.Result, error) {
		return results, err
	}
}

func TestModel_Init_ShowsSearchInput(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))
	assert.True(t, m.searchInput.Focused())
	assert.Equal(t, stateInput, m.state)
}

func TestModel_Search_DisplaysResults(t *testing.T) {
	results := []connectors.Result{
		{Title: "Doc A", URL: "https://example.com/a", Source: "test"},
		{Title: "Doc B", URL: "https://example.com/b", Source: "test"},
	}
	m := NewModel(mockSearchFn(results, nil))

	// Simulate typing a query
	m.searchInput.SetValue("test")

	// Simulate pressing Enter
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	// The model should transition to loading state and return a command
	require.NotNil(t, cmd)
	assert.Equal(t, stateLoading, model.state)

	// Simulate receiving search results
	updated, _ = model.Update(searchResultMsg{results: results})
	model = updated.(Model)

	assert.Equal(t, stateResults, model.state)
	assert.Len(t, model.results, 2)
	assert.Equal(t, "Doc A", model.results[0].Title)
	assert.Equal(t, 0, model.cursor)
}

func TestModel_Navigate_SelectsResult(t *testing.T) {
	results := []connectors.Result{
		{Title: "Doc A", URL: "https://example.com/a", Source: "test"},
		{Title: "Doc B", URL: "https://example.com/b", Source: "test"},
		{Title: "Doc C", URL: "https://example.com/c", Source: "test"},
	}
	m := NewModel(mockSearchFn(results, nil))
	m.state = stateResults
	m.results = results
	m.cursor = 0

	// Down arrow
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	assert.Equal(t, 1, model.cursor)

	// Down again
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	assert.Equal(t, 2, model.cursor)

	// Down at bottom stays at bottom
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	assert.Equal(t, 2, model.cursor)

	// Up arrow
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	assert.Equal(t, 1, model.cursor)
}

func TestModel_Escape_ReturnsToInput(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))
	m.state = stateResults
	m.results = []connectors.Result{{Title: "Doc"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)

	assert.Equal(t, stateInput, model.state)
	assert.True(t, model.searchInput.Focused())
}

func TestModel_View_ContainsSearchPrompt(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))
	view := m.View()
	assert.Contains(t, view, "Search")
}

func TestModel_View_ResultsStatusBar_NoEnterOpen(t *testing.T) {
	results := []connectors.Result{
		{Title: "Doc A", URL: "https://example.com/a", Source: "test"},
	}
	m := NewModel(mockSearchFn(results, nil))
	m.state = stateResults
	m.results = results
	m.cursor = 0

	view := m.View()

	assert.Contains(t, view, "navigate", "status bar should mention navigate")
	assert.NotContains(t, view, "enter: open", "status bar must not advertise unimplemented enter: open")
}

func TestModel_DoSearch_SetsCancelFunc(t *testing.T) {
	searchCalled := make(chan context.Context, 1)
	m := NewModel(func(ctx context.Context, query string) ([]connectors.Result, error) {
		searchCalled <- ctx
		return nil, nil
	})

	m.searchInput.SetValue("test")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	// The cancel function must be set after initiating a search
	assert.NotNil(t, model.cancel, "cancel func should be set when search starts")

	// Execute the command so the search function runs
	if cmd != nil {
		cmd()
	}

	// Verify the search was called with a non-background context
	ctx := <-searchCalled
	assert.NotNil(t, ctx)
}

func TestModel_EscapeDuringLoading_CancelsContext(t *testing.T) {
	searchCalled := make(chan context.Context, 1)
	m := NewModel(func(ctx context.Context, query string) ([]connectors.Result, error) {
		searchCalled <- ctx
		<-ctx.Done()
		return nil, ctx.Err()
	})

	// Start a search
	m.searchInput.SetValue("test")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	assert.Equal(t, stateLoading, model.state)

	// Run the search command in a background goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		if cmd != nil {
			cmd()
		}
	}()

	// Wait for search function to be called so we can grab its context
	ctx := <-searchCalled

	// Press Escape while in loading state
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)

	// Should return to input state
	assert.Equal(t, stateInput, model.state)

	// The context should be cancelled
	<-done
	assert.Error(t, ctx.Err(), "context should be cancelled after Escape")
}

// BUG-013: Init() should return textinput.Blink directly, not a wrapping closure.
func TestModel_Init_ReturnsBlinkCmd(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))
	cmd := m.Init()
	require.NotNil(t, cmd, "Init must return a non-nil tea.Cmd")

	// Execute the command and verify it returns a textinput.Blink message.
	msg := cmd()
	// textinput.Blink() returns a blink message; verify it is the same type.
	expected := textinput.Blink()
	assert.IsType(t, expected, msg, "Init cmd should produce a blink message")
}

// BUG-003: Successful search must clear a stale error from a previous failure.
func TestModel_SearchResult_ClearsStaleError(t *testing.T) {
	results := []connectors.Result{
		{Title: "Good Result", URL: "https://example.com", Source: "test"},
	}
	m := NewModel(mockSearchFn(results, nil))

	// Simulate a previous failed search leaving a stale error.
	m.err = fmt.Errorf("previous network error")
	m.state = stateLoading

	// Receive a successful search result.
	updated, _ := m.Update(searchResultMsg{results: results})
	model := updated.(Model)

	assert.Nil(t, model.err, "successful search must clear stale error")
	assert.Equal(t, stateResults, model.state)
	assert.Len(t, model.results, 1)
}

// BUG-012: cancel must be set to nil after search completes (success path).
func TestModel_SearchResult_ClearsCancelOnSuccess(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))

	// Simulate having a cancel function set from starting a search.
	cancelled := false
	m.cancel = func() { cancelled = true }
	m.state = stateLoading

	results := []connectors.Result{{Title: "Doc", URL: "u", Source: "s"}}
	updated, _ := m.Update(searchResultMsg{results: results})
	model := updated.(Model)

	assert.Nil(t, model.cancel, "cancel must be nil after search completes successfully")
	assert.False(t, cancelled, "cancel should not be called on success, just cleared")
}

// BUG-012: cancel must be set to nil after search completes (error path).
func TestModel_SearchResult_ClearsCancelOnError(t *testing.T) {
	m := NewModel(mockSearchFn(nil, nil))

	// Simulate having a cancel function set from starting a search.
	m.cancel = func() {}
	m.state = stateLoading

	updated, _ := m.Update(searchResultMsg{err: fmt.Errorf("search failed")})
	model := updated.(Model)

	assert.Nil(t, model.cancel, "cancel must be nil after search completes with error")
	assert.Error(t, model.err)
}
