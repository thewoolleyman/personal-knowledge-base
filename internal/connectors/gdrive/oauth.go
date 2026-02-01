package gdrive

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to a file as JSON.
func SaveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create token file: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

// LoadToken reads an OAuth2 token from a JSON file.
func LoadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open token file: %w", err)
	}
	defer f.Close()

	var tok oauth2.Token
	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &tok, nil
}
