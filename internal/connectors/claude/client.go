package claude

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxResults = 20

// Conversation represents a Claude.ai exported conversation.
type Conversation struct {
	UUID         string        `json:"uuid"`
	Name         string        `json:"name"`
	ChatMessages []ChatMessage `json:"chat_messages"`
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	UUID   string `json:"uuid"`
	Text   string `json:"text"`
	Sender string `json:"sender"`
}

// ClaudeClient abstracts access to Claude conversation data for testability.
type ClaudeClient interface {
	Search(ctx context.Context, query string) ([]Conversation, error)
}

// FileClient implements ClaudeClient by reading a JSON export from disk.
type FileClient struct {
	path string
}

// NewFileClient creates a client that reads from the given file path.
func NewFileClient(path string) *FileClient {
	return &FileClient{path: path}
}

func (c *FileClient) Search(_ context.Context, query string) ([]Conversation, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, fmt.Errorf("read claude export: %w", err)
	}

	var conversations []Conversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("decode claude export: %w", err)
	}

	queryLower := strings.ToLower(query)
	var matches []Conversation
	for _, conv := range conversations {
		if matchesQuery(conv, queryLower) {
			matches = append(matches, conv)
			if len(matches) >= maxResults {
				break
			}
		}
	}

	return matches, nil
}

func matchesQuery(conv Conversation, queryLower string) bool {
	if strings.Contains(strings.ToLower(conv.Name), queryLower) {
		return true
	}
	for _, msg := range conv.ChatMessages {
		if strings.Contains(strings.ToLower(msg.Text), queryLower) {
			return true
		}
	}
	return false
}

// ExtractConversationsJSON detects whether data is a zip archive or raw JSON.
// If zip, it extracts and returns the contents of conversations.json.
// If JSON, it returns data as-is.
func ExtractConversationsJSON(data []byte) ([]byte, error) {
	if isZip(data) {
		return extractFromZip(data)
	}
	return data, nil
}

func isZip(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x50 && data[1] == 0x4B
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}
	for _, f := range r.File {
		if f.Name == "conversations.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open conversations.json in zip: %w", err)
			}
			defer rc.Close()
			contents, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read conversations.json from zip: %w", err)
			}
			return contents, nil
		}
	}
	return nil, fmt.Errorf("zip archive does not contain conversations.json")
}

// userHomeDir returns the user's home directory. Overridden in tests.
var userHomeDir = os.UserHomeDir

// ExportUserHomeDir returns the current userHomeDir function (for test setup).
func ExportUserHomeDir() func() (string, error) { return userHomeDir }

// SetExportUserHomeDir overrides userHomeDir (for test setup).
func SetExportUserHomeDir(fn func() (string, error)) { userHomeDir = fn }

// DefaultExportPath returns the XDG-compliant default path for the Claude export.
// Uses $XDG_DATA_HOME/pkb/claude-conversations.json if set,
// otherwise ~/.local/share/pkb/claude-conversations.json.
func DefaultExportPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "pkb", "claude-conversations.json")
	}
	home, err := userHomeDir()
	if err != nil {
		return "claude-conversations.json"
	}
	return filepath.Join(home, ".local", "share", "pkb", "claude-conversations.json")
}
