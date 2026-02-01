package gmail

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGmailClient implements GmailClient for testing.
type MockGmailClient struct {
	mock.Mock
}

func (m *MockGmailClient) SearchMessages(ctx context.Context, query string) ([]Message, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]Message), args.Error(1)
}

func TestConnector_Name(t *testing.T) {
	c := NewConnector(nil)
	assert.Equal(t, "gmail", c.Name())
}

func TestConnector_Search_ReturnsResults(t *testing.T) {
	mockClient := new(MockGmailClient)
	mockClient.On("SearchMessages", mock.Anything, "test query").Return([]Message{
		{ID: "abc123", Subject: "Meeting Notes", Snippet: "Discussion about Q4 planning", From: "alice@example.com"},
		{ID: "def456", Subject: "Project Update", Snippet: "Sprint review summary", From: "bob@example.com"},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Meeting Notes", results[0].Title)
	assert.Equal(t, "Discussion about Q4 planning", results[0].Snippet)
	assert.Contains(t, results[0].URL, "abc123")
	assert.Equal(t, "gmail", results[0].Source)
	assert.Equal(t, "Project Update", results[1].Title)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesEmpty(t *testing.T) {
	mockClient := new(MockGmailClient)
	mockClient.On("SearchMessages", mock.Anything, "nothing").Return([]Message{}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "nothing")

	require.NoError(t, err)
	assert.Empty(t, results)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesError(t *testing.T) {
	mockClient := new(MockGmailClient)
	mockClient.On("SearchMessages", mock.Anything, "fail").Return([]Message(nil), errors.New("API rate limit"))

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "fail")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "API rate limit")
	mockClient.AssertExpectations(t)
}
