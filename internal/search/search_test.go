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
