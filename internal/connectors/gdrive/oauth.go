package gdrive

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// SaveToken writes an OAuth2 token to a file as JSON.
// It creates the parent directory if it does not exist.
func SaveToken(path string, token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create token file: %w", err)
	}
	return encodeAndClose(f, token)
}

// encodeAndClose writes a token as JSON and closes the writer,
// surfacing both encode and close errors.
func encodeAndClose(wc io.WriteCloser, token *oauth2.Token) error {
	err := json.NewEncoder(wc).Encode(token)
	if closeErr := wc.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("save token file: %w", err)
	}
	return nil
}

// PersistingTokenSource wraps an oauth2.TokenSource and writes the token back
// to disk whenever it is refreshed. This ensures rotated refresh tokens are not
// lost between runs.
type PersistingTokenSource struct {
	src  oauth2.TokenSource
	path string
	last *oauth2.Token
}

// NewPersistingTokenSource creates a token source that persists refreshed tokens.
func NewPersistingTokenSource(src oauth2.TokenSource, path string, initial *oauth2.Token) *PersistingTokenSource {
	return &PersistingTokenSource{src: src, path: path, last: initial}
}

// Token returns a valid token, saving it to disk if it was refreshed.
func (p *PersistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		return nil, err
	}
	// If the token changed (refreshed), persist it.
	if tok.AccessToken != p.last.AccessToken {
		_ = SaveToken(p.path, tok) // best-effort; don't fail the request
		p.last = tok
	}
	return tok, nil
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
