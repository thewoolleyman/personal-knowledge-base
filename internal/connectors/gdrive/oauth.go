package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// SaveTokenAtomic writes an OAuth2 token to a file atomically by writing to a
// temporary file in the same directory and then renaming it into place.
// The write, chmod, and rename steps are done via package-level functions
// that can be overridden in tests to simulate errors.
func SaveTokenAtomic(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	tmp, err := osCreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp token file: %w", err)
	}
	tmpName := tmp.Name()

	if err := encodeAndClose(tmp, token); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := osChmod(tmpName, 0600); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("chmod token file: %w", err)
	}
	if err := osRename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename token file: %w", err)
	}
	return nil
}

// OS functions overridden in tests to simulate errors.
var (
	osCreateTemp = os.CreateTemp
	osChmod      = os.Chmod
	osRename     = os.Rename
)

// TokenChecker provides health information about the current token.
type TokenChecker interface {
	TokenHealth() TokenStatus
}

// TokenStatus describes the health of an OAuth2 token.
type TokenStatus struct {
	Valid     bool
	ExpiresIn time.Duration
	Error     string
}

// PersistingTokenSource wraps an oauth2.TokenSource and writes the token back
// to disk whenever it is refreshed. This ensures rotated refresh tokens are not
// lost between runs.
type PersistingTokenSource struct {
	src  oauth2.TokenSource
	path string
	last *oauth2.Token

	// SaveFunc is the function used to persist tokens. If nil, defaults to
	// SaveTokenAtomic.
	SaveFunc func(string, *oauth2.Token) error

	// Logger for token refresh events. If nil, defaults to slog.Default().
	Logger *slog.Logger
}

// NewPersistingTokenSource creates a token source that persists refreshed tokens.
func NewPersistingTokenSource(src oauth2.TokenSource, path string, initial *oauth2.Token) *PersistingTokenSource {
	return &PersistingTokenSource{src: src, path: path, last: initial}
}

func (p *PersistingTokenSource) logger() *slog.Logger {
	if p.Logger != nil {
		return p.Logger
	}
	return slog.Default()
}

func (p *PersistingTokenSource) saveFunc() func(string, *oauth2.Token) error {
	if p.SaveFunc != nil {
		return p.SaveFunc
	}
	return SaveTokenAtomic
}

// Token returns a valid token, saving it to disk if it was refreshed.
func (p *PersistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") ||
			strings.Contains(err.Error(), "Token has been expired or revoked") {
			return nil, fmt.Errorf("OAuth refresh token is invalid or expired. Run 'pkb auth' to re-authenticate: %w", err)
		}
		return nil, err
	}

	// Warn if token expiry is within 10 minutes.
	if !tok.Expiry.IsZero() && time.Until(tok.Expiry) < 10*time.Minute {
		p.logger().Warn("token expiry is within 10 minutes",
			"expires_in", time.Until(tok.Expiry).Round(time.Second))
	}

	// If the token changed (refreshed), persist it.
	if tok.AccessToken != p.last.AccessToken {
		p.logger().Info("token refreshed successfully")
		save := p.saveFunc()
		var saveErr error
		for attempt := 0; attempt < 3; attempt++ {
			saveErr = save(p.path, tok)
			if saveErr == nil {
				break
			}
		}
		if saveErr != nil {
			p.logger().Error("failed to persist refreshed token after retries", "error", saveErr)
		}
		p.last = tok
	}
	return tok, nil
}

// StartBackgroundRefresh starts a goroutine that proactively refreshes the token
// before it expires. It checks the token expiry at the given interval and triggers
// a refresh if the token is within 10 minutes of expiry. The goroutine stops when
// the context is cancelled.
func (p *PersistingTokenSource) StartBackgroundRefresh(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if p.last != nil && !p.last.Expiry.IsZero() && time.Until(p.last.Expiry) < 10*time.Minute {
					_, _ = p.Token()
				}
			}
		}
	}()
}

// TokenHealth returns the current health status of the token.
func (p *PersistingTokenSource) TokenHealth() TokenStatus {
	if p.last == nil {
		return TokenStatus{Valid: false, Error: "no token available"}
	}
	if p.last.Expiry.IsZero() {
		return TokenStatus{Valid: true}
	}
	expiresIn := time.Until(p.last.Expiry)
	if expiresIn <= 0 {
		return TokenStatus{
			Valid:     false,
			ExpiresIn: expiresIn,
			Error:     "token expired",
		}
	}
	return TokenStatus{
		Valid:     true,
		ExpiresIn: expiresIn,
	}
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
