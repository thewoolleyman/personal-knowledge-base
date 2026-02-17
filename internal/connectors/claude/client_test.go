package claude

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileClient_Search_MatchesConversationName(t *testing.T) {
	path := writeTempJSON(t, `[
		{
			"uuid": "conv-1",
			"name": "Debugging Go tests",
			"chat_messages": [
				{"uuid": "msg-1", "text": "Help me debug", "sender": "human"}
			]
		},
		{
			"uuid": "conv-2",
			"name": "Python scripting",
			"chat_messages": [
				{"uuid": "msg-2", "text": "Write a script", "sender": "human"}
			]
		}
	]`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "go tests")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "conv-1", results[0].UUID)
	assert.Equal(t, "Debugging Go tests", results[0].Name)
}

func TestFileClient_Search_MatchesMessageText(t *testing.T) {
	path := writeTempJSON(t, `[
		{
			"uuid": "conv-1",
			"name": "General chat",
			"chat_messages": [
				{"uuid": "msg-1", "text": "How do I deploy to kubernetes?", "sender": "human"},
				{"uuid": "msg-2", "text": "Use kubectl apply...", "sender": "assistant"}
			]
		}
	]`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "kubernetes")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "conv-1", results[0].UUID)
}

func TestFileClient_Search_CaseInsensitive(t *testing.T) {
	path := writeTempJSON(t, `[
		{
			"uuid": "conv-1",
			"name": "Docker Setup",
			"chat_messages": [{"uuid": "msg-1", "text": "help", "sender": "human"}]
		}
	]`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "docker setup")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "conv-1", results[0].UUID)
}

func TestFileClient_Search_NoMatch(t *testing.T) {
	path := writeTempJSON(t, `[
		{
			"uuid": "conv-1",
			"name": "Cooking recipes",
			"chat_messages": [{"uuid": "msg-1", "text": "pasta recipe", "sender": "human"}]
		}
	]`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "kubernetes")

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFileClient_Search_EmptyArray(t *testing.T) {
	path := writeTempJSON(t, `[]`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "anything")

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFileClient_Search_MissingFile(t *testing.T) {
	client := NewFileClient("/nonexistent/path/conversations.json")
	results, err := client.Search(context.Background(), "query")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "read claude export")
}

func TestFileClient_Search_MalformedJSON(t *testing.T) {
	path := writeTempJSON(t, `{not valid json`)

	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "query")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "decode claude export")
}

func TestFileClient_Search_LimitsTo20Results(t *testing.T) {
	// Build JSON with 25 conversations all matching "match"
	conversations := "["
	for i := 0; i < 25; i++ {
		if i > 0 {
			conversations += ","
		}
		conversations += `{"uuid":"conv-` + string(rune('A'+i)) + `","name":"match item","chat_messages":[]}`
	}
	conversations += "]"

	path := writeTempJSON(t, conversations)
	client := NewFileClient(path)
	results, err := client.Search(context.Background(), "match")

	require.NoError(t, err)
	assert.Len(t, results, 20)
}

func TestDefaultExportPath_UsesXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/test-xdg-data")
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })

	path := DefaultExportPath()
	assert.Equal(t, filepath.Join("/tmp/test-xdg-data", "pkb", "claude-conversations.json"), path)
}

func TestDefaultExportPath_FallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "/tmp/test-home", nil }
	t.Cleanup(func() { userHomeDir = orig })

	path := DefaultExportPath()
	assert.Equal(t, filepath.Join("/tmp/test-home", ".local", "share", "pkb", "claude-conversations.json"), path)
}

func TestDefaultExportPath_FallsBackToFilename_WhenHomeDirFails(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	t.Cleanup(func() { userHomeDir = orig })

	path := DefaultExportPath()
	assert.Equal(t, "claude-conversations.json", path)
}

func TestExtractConversationsJSON_RawJSON(t *testing.T) {
	input := []byte(`[{"uuid":"conv-1","name":"Test","chat_messages":[]}]`)
	got, err := ExtractConversationsJSON(input)
	require.NoError(t, err)
	assert.Equal(t, input, got)
}

func TestExtractConversationsJSON_ZipWithConversations(t *testing.T) {
	jsonContent := `[{"uuid":"conv-1","name":"Zip Test","chat_messages":[]}]`
	zipData := createTestZip(t, map[string]string{
		"conversations.json": jsonContent,
		"projects.json":      "[]",
		"memories.json":      "[]",
	})

	got, err := ExtractConversationsJSON(zipData)
	require.NoError(t, err)
	assert.Equal(t, jsonContent, string(got))
}

func TestExtractConversationsJSON_ZipMissingConversations(t *testing.T) {
	zipData := createTestZip(t, map[string]string{
		"projects.json": "[]",
	})

	_, err := ExtractConversationsJSON(zipData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversations.json")
}

func TestExtractConversationsJSON_InvalidZip(t *testing.T) {
	// PK magic bytes but not a valid zip
	input := []byte{0x50, 0x4B, 0x00, 0x00, 0xFF, 0xFF}
	_, err := ExtractConversationsJSON(input)
	assert.Error(t, err)
}

// createTestZip creates a zip archive in memory with the given files.
func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// writeTempJSON writes content to a temp file and returns the path.
func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "conversations.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}
