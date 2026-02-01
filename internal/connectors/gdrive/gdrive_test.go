package gdrive

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDriveClient implements DriveClient for testing.
type MockDriveClient struct {
	mock.Mock
}

func (m *MockDriveClient) SearchFiles(ctx context.Context, query string) ([]DriveFile, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]DriveFile), args.Error(1)
}

func TestConnector_Name(t *testing.T) {
	c := NewConnector(nil)
	assert.Equal(t, "google-drive", c.Name())
}

func TestConnector_Search_ReturnsResults(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "test query").Return([]DriveFile{
		{ID: "abc123", Name: "Meeting Notes.md", MimeType: "text/markdown", WebViewLink: "https://drive.google.com/file/d/abc123/view", Description: "Weekly meeting notes"},
		{ID: "def456", Name: "Project Plan.docx", MimeType: "application/vnd.google-apps.document", WebViewLink: "https://drive.google.com/file/d/def456/view", Description: "Q1 project plan"},
	}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Meeting Notes.md", results[0].Title)
	assert.Equal(t, "https://drive.google.com/file/d/abc123/view", results[0].URL)
	assert.Equal(t, "google-drive", results[0].Source)
	assert.Equal(t, "Weekly meeting notes", results[0].Snippet)
	assert.Equal(t, "Project Plan.docx", results[1].Title)
	assert.Equal(t, "Q1 project plan", results[1].Snippet)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesEmpty(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "nothing").Return([]DriveFile{}, nil)

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "nothing")

	require.NoError(t, err)
	assert.Empty(t, results)
	mockClient.AssertExpectations(t)
}

func TestConnector_Search_HandlesError(t *testing.T) {
	mockClient := new(MockDriveClient)
	mockClient.On("SearchFiles", mock.Anything, "fail").Return([]DriveFile(nil), errors.New("API rate limit"))

	c := NewConnector(mockClient)
	results, err := c.Search(context.Background(), "fail")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "API rate limit")
	mockClient.AssertExpectations(t)
}
