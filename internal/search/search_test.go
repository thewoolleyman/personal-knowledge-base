package search

import (
	"context"
	"errors"
	"testing"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConnector implements connectors.Connector for testing.
type MockConnector struct {
	mock.Mock
}

func (m *MockConnector) Search(ctx context.Context, query string) ([]connectors.Result, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]connectors.Result), args.Error(1)
}

func (m *MockConnector) Name() string {
	args := m.Called()
	return args.String(0)
}

func TestEngine_Search_FansOutToConnectors(t *testing.T) {
	mock1 := new(MockConnector)
	mock2 := new(MockConnector)

	mock1.On("Search", mock.Anything, "test query").Return([]connectors.Result{
		{Title: "Result A", Source: "mock1"},
	}, nil)
	mock1.On("Name").Return("mock1")

	mock2.On("Search", mock.Anything, "test query").Return([]connectors.Result{
		{Title: "Result B", Source: "mock2"},
	}, nil)
	mock2.On("Name").Return("mock2")

	engine := New(mock1, mock2)
	results, err := engine.Search(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, results, 2)

	titles := []string{results[0].Title, results[1].Title}
	assert.ElementsMatch(t, []string{"Result A", "Result B"}, titles)
	mock1.AssertExpectations(t)
	mock2.AssertExpectations(t)
}

func TestEngine_Search_HandlesPartialFailure(t *testing.T) {
	mock1 := new(MockConnector)
	mock2 := new(MockConnector)

	mock1.On("Search", mock.Anything, "test").Return([]connectors.Result{
		{Title: "Good Result"},
	}, nil)
	mock1.On("Name").Return("mock1")

	mock2.On("Search", mock.Anything, "test").Return([]connectors.Result(nil), errors.New("connection refused"))
	mock2.On("Name").Return("mock2")

	engine := New(mock1, mock2)
	results, err := engine.Search(context.Background(), "test")

	// Partial failure: still returns results from healthy connectors, no error
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Good Result", results[0].Title)
}

func TestEngine_Search_NoConnectors(t *testing.T) {
	engine := New()
	results, err := engine.Search(context.Background(), "test")

	require.NoError(t, err)
	assert.Empty(t, results)
	// BUG-014: Must return non-nil empty slice, not nil.
	assert.NotNil(t, results, "expected non-nil empty slice for zero connectors")
}

func TestEngine_Search_AllFail(t *testing.T) {
	mock1 := new(MockConnector)
	mock1.On("Search", mock.Anything, "q").Return([]connectors.Result(nil), errors.New("fail"))
	mock1.On("Name").Return("mock1")

	engine := New(mock1)
	results, err := engine.Search(context.Background(), "q")

	// All failed: returns error
	assert.Error(t, err)
	assert.Empty(t, results)
}

func TestEngine_SearchWithSources_FiltersConnectors(t *testing.T) {
	drive := new(MockConnector)
	drive.On("Name").Return("gdrive")
	drive.On("Search", mock.Anything, "q").Return([]connectors.Result{
		{Title: "Drive Doc", Source: "gdrive"},
	}, nil)

	gm := new(MockConnector)
	gm.On("Name").Return("gmail")
	// gmail.Search should NOT be called

	engine := New(drive, gm)
	results, err := engine.SearchWithSources(context.Background(), "q", []string{"gdrive"})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Drive Doc", results[0].Title)
	drive.AssertExpectations(t)
	gm.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
}

func TestEngine_SearchWithSources_NilSearchesAll(t *testing.T) {
	drive := new(MockConnector)
	drive.On("Name").Return("gdrive")
	drive.On("Search", mock.Anything, "q").Return([]connectors.Result{
		{Title: "Drive Doc", Source: "gdrive"},
	}, nil)

	gm := new(MockConnector)
	gm.On("Name").Return("gmail")
	gm.On("Search", mock.Anything, "q").Return([]connectors.Result{
		{Title: "Email", Source: "gmail"},
	}, nil)

	engine := New(drive, gm)
	results, err := engine.SearchWithSources(context.Background(), "q", nil)

	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestEngine_SearchWithSources_EmptySearchesAll(t *testing.T) {
	drive := new(MockConnector)
	drive.On("Name").Return("gdrive")
	drive.On("Search", mock.Anything, "q").Return([]connectors.Result{
		{Title: "Drive Doc", Source: "gdrive"},
	}, nil)

	engine := New(drive)
	results, err := engine.SearchWithSources(context.Background(), "q", []string{})

	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestEngine_SearchWithSources_UnknownSourceIgnored(t *testing.T) {
	drive := new(MockConnector)
	drive.On("Name").Return("gdrive")
	// drive.Search should NOT be called since "nonexistent" doesn't match

	engine := New(drive)
	results, err := engine.SearchWithSources(context.Background(), "q", []string{"nonexistent"})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestEngine_ConnectorNames(t *testing.T) {
	drive := new(MockConnector)
	drive.On("Name").Return("gdrive")
	gm := new(MockConnector)
	gm.On("Name").Return("gmail")

	engine := New(drive, gm)
	names := engine.ConnectorNames()
	assert.ElementsMatch(t, []string{"gdrive", "gmail"}, names)
}

func TestEngine_ConnectorNames_Empty(t *testing.T) {
	engine := New()
	names := engine.ConnectorNames()
	assert.Empty(t, names)
}
