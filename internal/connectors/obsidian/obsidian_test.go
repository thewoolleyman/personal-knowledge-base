package obsidian

import (
	"context"
	"errors"
	"testing"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDriveClient implements gdrive.DriveClient for testing.
type MockDriveClient struct {
	mock.Mock
}

func (m *MockDriveClient) SearchFiles(ctx context.Context, query string) ([]gdrive.DriveFile, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]gdrive.DriveFile), args.Error(1)
}

func TestConnector_Name(t *testing.T) {
	c := NewConnector(nil, "folder-123")
	assert.Equal(t, "obsidian", c.Name())
}

func TestConnector_Search_ReturnsResults(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "meeting notes").Return([]gdrive.DriveFile{
		{ID: "abc123", Name: "Daily Note.md", MimeType: "text/markdown", WebViewLink: "https://drive.google.com/file/d/abc123/view", Description: "Daily journal entry"},
		{ID: "def456", Name: "Project Ideas.md", MimeType: "text/markdown", WebViewLink: "https://drive.google.com/file/d/def456/view", Description: "Brainstorming ideas"},
	}, nil)

	c := NewConnector(mockClient, "folder-123")
	results, err := c.Search(context.Background(), "meeting notes")

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Daily Note.md", results[0].Title)
	assert.Equal(t, "https://drive.google.com/file/d/abc123/view", results[0].URL)
	assert.Equal(t, "obsidian", results[0].Source)
	assert.Equal(t, "Daily journal entry", results[0].Snippet)
	assert.Equal(t, "Project Ideas.md", results[1].Title)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesEmpty(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "nothing").Return([]gdrive.DriveFile{}, nil)

	c := NewConnector(mockClient, "folder-123")
	results, err := c.Search(context.Background(), "nothing")

	require.NoError(t, err)
	assert.Empty(t, results)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesError(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "fail").Return([]gdrive.DriveFile(nil), errors.New("API rate limit"))

	c := NewConnector(mockClient, "folder-123")
	results, err := c.Search(context.Background(), "fail")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "obsidian search")
	assert.Contains(t, err.Error(), "API rate limit")
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_SourceIsObsidian(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "test").Return([]gdrive.DriveFile{
		{ID: "1", Name: "Note.md", WebViewLink: "https://example.com"},
	}, nil)

	c := NewConnector(mockClient, "folder-123")
	results, err := c.Search(context.Background(), "test")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "obsidian", results[0].Source, "Source must be 'obsidian', not 'google-drive'")
	mockClient.AssertExpectations(t)
}
