package claude

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClaudeClient implements ClaudeClient for testing.
type MockClaudeClient struct {
	mock.Mock
}

func (m *MockClaudeClient) Search(ctx context.Context, query string) ([]Conversation, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]Conversation), args.Error(1)
}

func TestConnector_Name(t *testing.T) {
	c := NewConnector(nil)
	assert.Equal(t, "claude", c.Name())
}

func TestConnector_Search_ReturnsResults(t *testing.T) {
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "test query").Return([]Conversation{
		{
			UUID: "conv-abc-123",
			Name: "Debugging test failures",
			ChatMessages: []ChatMessage{
				{UUID: "msg-1", Text: "How do I fix test query failures?", Sender: "human"},
				{UUID: "msg-2", Text: "Here are some approaches...", Sender: "assistant"},
			},
		},
		{
			UUID: "conv-def-456",
			Name: "Code review discussion",
			ChatMessages: []ChatMessage{
				{UUID: "msg-3", Text: "Please review this test query code", Sender: "human"},
			},
		},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Debugging test failures", results[0].Title)
	assert.Equal(t, "https://claude.ai/chat/conv-abc-123", results[0].URL)
	assert.Equal(t, "claude", results[0].Source)
	assert.NotEmpty(t, results[0].Snippet)
	assert.Equal(t, "Code review discussion", results[1].Title)
	assert.Equal(t, "https://claude.ai/chat/conv-def-456", results[1].URL)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_SnippetFromFirstMatchingMessage(t *testing.T) {
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "deploy").Return([]Conversation{
		{
			UUID: "conv-1",
			Name: "Deployment help",
			ChatMessages: []ChatMessage{
				{UUID: "msg-1", Text: "How do I deploy to production?", Sender: "human"},
				{UUID: "msg-2", Text: "You can deploy using Docker...", Sender: "assistant"},
			},
		},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "deploy")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Snippet, "deploy")
}

func TestConnector_Search_SnippetTruncation(t *testing.T) {
	longText := ""
	for i := 0; i < 300; i++ {
		longText += "x"
	}
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "x").Return([]Conversation{
		{
			UUID:         "conv-1",
			Name:         "Long conversation",
			ChatMessages: []ChatMessage{{UUID: "msg-1", Text: longText, Sender: "human"}},
		},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "x")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.LessOrEqual(t, len(results[0].Snippet), 203) // 200 + "..."
}

func TestConnector_Search_SnippetFallsBackToFirstMessage_WhenNameMatches(t *testing.T) {
	// The query matches the conversation name but NOT any message text.
	// Snippet should fall back to the first message.
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "architecture").Return([]Conversation{
		{
			UUID: "conv-1",
			Name: "Architecture review",
			ChatMessages: []ChatMessage{
				{UUID: "msg-1", Text: "Let's discuss the design", Sender: "human"},
			},
		},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "architecture")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Let's discuss the design", results[0].Snippet)
}

func TestConnector_Search_SnippetEmpty_WhenNoMessages(t *testing.T) {
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "empty").Return([]Conversation{
		{
			UUID:         "conv-1",
			Name:         "empty chat",
			ChatMessages: nil,
		},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "empty")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Empty(t, results[0].Snippet)
}

func TestExportUserHomeDir_RoundTrip(t *testing.T) {
	orig := ExportUserHomeDir()
	t.Cleanup(func() { SetExportUserHomeDir(orig) })

	custom := func() (string, error) { return "/custom", nil }
	SetExportUserHomeDir(custom)

	// Verify the override took effect via DefaultExportPath.
	t.Setenv("XDG_DATA_HOME", "")
	path := DefaultExportPath()
	assert.Contains(t, path, "/custom")
}

func TestConnector_Search_HandlesEmpty(t *testing.T) {
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "nothing").Return([]Conversation{}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "nothing")

	require.NoError(t, err)
	assert.Empty(t, results)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesError(t *testing.T) {
	mockClient := new(MockClaudeClient)
	mockClient.On("Search", mock.Anything, "fail").Return([]Conversation(nil), errors.New("file not found"))

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "fail")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "claude search")
	assert.Contains(t, err.Error(), "file not found")
	mockClient.AssertExpectations(t)
}
