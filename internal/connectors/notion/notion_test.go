package notion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockNotionClient implements NotionClient for testing.
type MockNotionClient struct {
	mock.Mock
}

func (m *MockNotionClient) Search(ctx context.Context, query string) ([]Page, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]Page), args.Error(1)
}

func TestConnector_Name(t *testing.T) {
	c := NewConnector(nil)
	assert.Equal(t, "notion", c.Name())
}

func TestConnector_Search_ReturnsResults(t *testing.T) {
	mockClient := new(MockNotionClient)
	mockClient.On("Search", mock.Anything, "test query").Return([]Page{
		{ID: "page-1", Title: "Meeting Notes", URL: "https://www.notion.so/Meeting-Notes-abc123"},
		{ID: "page-2", Title: "Project Plan", URL: "https://www.notion.so/Project-Plan-def456"},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Meeting Notes", results[0].Title)
	assert.Equal(t, "https://www.notion.so/Meeting-Notes-abc123", results[0].URL)
	assert.Equal(t, "notion", results[0].Source)
	assert.Equal(t, "Project Plan", results[1].Title)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesEmpty(t *testing.T) {
	mockClient := new(MockNotionClient)
	mockClient.On("Search", mock.Anything, "nothing").Return([]Page{}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "nothing")

	require.NoError(t, err)
	assert.Empty(t, results)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesError(t *testing.T) {
	mockClient := new(MockNotionClient)
	mockClient.On("Search", mock.Anything, "fail").Return([]Page(nil), errors.New("API rate limit"))

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "fail")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "notion search")
	assert.Contains(t, err.Error(), "API rate limit")
	mockClient.AssertExpectations(t)
}
