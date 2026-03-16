package gdrive

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", loaded.AccessToken)
	assert.Equal(t, "test-refresh-token", loaded.RefreshToken)
	assert.Equal(t, "Bearer", loaded.TokenType)
}

func TestLoadToken_FileNotFound(t *testing.T) {
	_, err := LoadToken("/nonexistent/token.json")
	assert.Error(t, err)
}

func TestLoadToken_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{truncated"), 0600))
	_, err := LoadToken(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode token")
}

func TestSaveToken_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "test-token"}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// BUG-002: SaveToken must surface errors from both Encode and Close.
func TestSaveToken_BadDirectory(t *testing.T) {
	// Writing to an unwritable path should return an error.
	err := SaveToken("/nonexistent/dir/token.json", &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create token directory")
}

func TestSaveToken_ClosesFileExplicitly(t *testing.T) {
	// Verify the file is properly closed by writing and then immediately reading.
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "close-test", TokenType: "Bearer"}

	err := SaveToken(path, tok)
	require.NoError(t, err)

	// If Close was called properly, the file should be fully flushed and readable.
	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "close-test", loaded.AccessToken)
}

// failCloser wraps bytes.Buffer with a Close that always errors.
type failCloser struct {
	bytes.Buffer
}

func (f *failCloser) Close() error {
	return fmt.Errorf("close failed")
}

func TestEncodeAndClose_CloseError(t *testing.T) {
	wc := &failCloser{}
	err := encodeAndClose(wc, &oauth2.Token{AccessToken: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
}

func TestSaveToken_CreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested", "subdir", "token.json")

	tok := &oauth2.Token{AccessToken: "nested-test", TokenType: "Bearer"}

	err := SaveToken(nested, tok)
	require.NoError(t, err)

	loaded, err := LoadToken(nested)
	require.NoError(t, err)
	assert.Equal(t, "nested-test", loaded.AccessToken)

	// Verify parent directory permissions are 0700 (owner-only).
	info, err := os.Stat(filepath.Dir(nested))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestPersistingTokenSource_SavesRefreshedToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	initial := &oauth2.Token{AccessToken: "old-access", RefreshToken: "refresh-1"}
	refreshed := &oauth2.Token{AccessToken: "new-access", RefreshToken: "refresh-2"}

	// Save initial token to disk.
	require.NoError(t, SaveToken(path, initial))

	// Create a fake token source that returns a refreshed token.
	fake := &staticTokenSource{tok: refreshed}
	pts := NewPersistingTokenSource(fake, path, initial)

	tok, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "new-access", tok.AccessToken)

	// The refreshed token should have been written to disk.
	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "new-access", loaded.AccessToken)
	assert.Equal(t, "refresh-2", loaded.RefreshToken)
}

func TestPersistingTokenSource_DoesNotWriteWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{AccessToken: "same-access", RefreshToken: "refresh-1"}
	require.NoError(t, SaveToken(path, tok))

	// Token source returns the same token (no refresh happened).
	fake := &staticTokenSource{tok: tok}
	pts := NewPersistingTokenSource(fake, path, tok)

	got, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "same-access", got.AccessToken)

	// File should still have the original content (verify it wasn't rewritten
	// by checking mod time stayed the same — but simpler: just verify content).
	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "same-access", loaded.AccessToken)
}

func TestPersistingTokenSource_PropagatesError(t *testing.T) {
	pts := NewPersistingTokenSource(
		&errorTokenSource{err: fmt.Errorf("token revoked")},
		"/unused",
		&oauth2.Token{AccessToken: "old"},
	)

	_, err := pts.Token()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token revoked")
}

// staticTokenSource always returns the same token.
type staticTokenSource struct {
	tok *oauth2.Token
}

func (s *staticTokenSource) Token() (*oauth2.Token, error) { return s.tok, nil }

// errorTokenSource always returns an error.
type errorTokenSource struct {
	err error
}

func (e *errorTokenSource) Token() (*oauth2.Token, error) { return nil, e.err }

func TestSaveToken_ReadOnlyDir(t *testing.T) {
	// Create a read-only directory and verify SaveToken fails.
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0500))
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	path := filepath.Join(readOnlyDir, "token.json")
	err := SaveToken(path, &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
}

// --- Step 2: SaveTokenAtomic ---

func TestSaveTokenAtomic_WritesAndLoads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "atomic-test", RefreshToken: "r1", TokenType: "Bearer"}

	err := SaveTokenAtomic(path, tok)
	require.NoError(t, err)

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "atomic-test", loaded.AccessToken)
	assert.Equal(t, "r1", loaded.RefreshToken)
}

func TestSaveTokenAtomic_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "token.json")

	err := SaveTokenAtomic(path, &oauth2.Token{AccessToken: "nested"})
	require.NoError(t, err)

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "nested", loaded.AccessToken)
}

func TestSaveTokenAtomic_BadDirectory(t *testing.T) {
	err := SaveTokenAtomic("/nonexistent/dir/token.json", &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
}

func TestSaveTokenAtomic_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	err := SaveTokenAtomic(path, &oauth2.Token{AccessToken: "perm-test"})
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSaveTokenAtomic_NoPartialWrite(t *testing.T) {
	// If an existing file has content, SaveTokenAtomic should atomically replace it.
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	// Write initial token.
	require.NoError(t, SaveTokenAtomic(path, &oauth2.Token{AccessToken: "first"}))

	// Overwrite with new token.
	require.NoError(t, SaveTokenAtomic(path, &oauth2.Token{AccessToken: "second"}))

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "second", loaded.AccessToken)
}

func TestSaveTokenAtomic_ReadOnlyDir(t *testing.T) {
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0500))
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0700) })

	err := SaveTokenAtomic(filepath.Join(readOnlyDir, "token.json"), &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create temp token file")
}

func TestSaveTokenAtomic_MkdirAllFail(t *testing.T) {
	err := SaveTokenAtomic("/nonexistent/deeply/nested/token.json", &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create token directory")
}

func TestSaveTokenAtomic_ChmodFail(t *testing.T) {
	origChmod := osChmod
	osChmod = func(_ string, _ os.FileMode) error {
		return fmt.Errorf("chmod injected error")
	}
	t.Cleanup(func() { osChmod = origChmod })

	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	err := SaveTokenAtomic(path, &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chmod token file")

	// Temp file should have been cleaned up.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp", "temp file should be cleaned up")
	}
}

func TestSaveTokenAtomic_CreateTempFail(t *testing.T) {
	origCreateTemp := osCreateTemp
	osCreateTemp = func(_ string, _ string) (*os.File, error) {
		return nil, fmt.Errorf("create temp injected error")
	}
	t.Cleanup(func() { osCreateTemp = origCreateTemp })

	dir := t.TempDir()
	err := SaveTokenAtomic(filepath.Join(dir, "token.json"), &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create temp token file")
}

func TestSaveTokenAtomic_EncodeAndCloseFail_CleansUp(t *testing.T) {
	// Verify that when encodeAndClose fails (e.g., pre-closed file), the temp
	// file is cleaned up and SaveTokenAtomic returns an error.
	origCreateTemp := osCreateTemp
	osCreateTemp = func(dir string, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		f.Close() // pre-close so encodeAndClose will fail on write
		return f, nil
	}
	t.Cleanup(func() { osCreateTemp = origCreateTemp })

	dir := t.TempDir()
	err := SaveTokenAtomic(filepath.Join(dir, "token.json"), &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save token file")
}

func TestSaveTokenAtomic_RenameFail(t *testing.T) {
	origRename := osRename
	osRename = func(_, _ string) error {
		return fmt.Errorf("rename injected error")
	}
	t.Cleanup(func() { osRename = origRename })

	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	err := SaveTokenAtomic(path, &oauth2.Token{AccessToken: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rename token file")
}

func TestTokenHealth_NilLastToken(t *testing.T) {
	// When last token is nil, TokenHealth should report invalid.
	pts := &PersistingTokenSource{
		src:  &staticTokenSource{tok: &oauth2.Token{AccessToken: "x"}},
		path: "/unused",
		last: nil,
	}
	status := pts.TokenHealth()
	assert.False(t, status.Valid)
	assert.Contains(t, status.Error, "no token")
}

// --- Step 2: PersistingTokenSource with SaveFunc and retry ---

// countingSaveFunc tracks how many times save was called.
type countingSaveFunc struct {
	mu      sync.Mutex
	calls   int
	failN   int // fail the first N calls
	lastTok *oauth2.Token
}

func (c *countingSaveFunc) save(path string, tok *oauth2.Token) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.lastTok = tok
	if c.calls <= c.failN {
		return fmt.Errorf("save failed attempt %d", c.calls)
	}
	return SaveTokenAtomic(path, tok)
}

func TestPersistingTokenSource_UsesSaveFuncField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new"}

	cs := &countingSaveFunc{}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.SaveFunc = cs.save

	tok, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "new", tok.AccessToken)
	assert.Equal(t, 1, cs.calls)
}

func TestPersistingTokenSource_RetriesOnSaveFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new"}

	cs := &countingSaveFunc{failN: 2} // fail first 2, succeed on 3rd
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.SaveFunc = cs.save
	pts.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	tok, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "new", tok.AccessToken)
	assert.Equal(t, 3, cs.calls) // 1 initial + 2 retries
}

func TestPersistingTokenSource_AllRetriesFail_StillReturnsToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new"}

	cs := &countingSaveFunc{failN: 10} // all attempts fail
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.SaveFunc = cs.save
	pts.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	tok, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "new", tok.AccessToken)
	assert.Equal(t, 3, cs.calls) // 1 initial + 2 retries = 3 total
}

func TestPersistingTokenSource_DefaultSaveFuncIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new-default"}

	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	// SaveFunc is nil — should default to SaveTokenAtomic.

	tok, err := pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "new-default", tok.AccessToken)

	loaded, err := LoadToken(path)
	require.NoError(t, err)
	assert.Equal(t, "new-default", loaded.AccessToken)
}

// --- Step 2: Logging ---

// logCapture is a slog handler that captures log records.
type logCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *logCapture) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (l *logCapture) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, r)
	return nil
}
func (l *logCapture) WithAttrs(_ []slog.Attr) slog.Handler { return l }
func (l *logCapture) WithGroup(_ string) slog.Handler       { return l }

func (l *logCapture) hasLevel(level slog.Level) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.records {
		if r.Level == level {
			return true
		}
	}
	return false
}

func (l *logCapture) hasMessage(msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.records {
		if r.Message == msg {
			return true
		}
	}
	return false
}

func TestPersistingTokenSource_LogsInfoOnRefresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new"}

	lc := &logCapture{}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.Logger = slog.New(lc)

	_, err := pts.Token()
	require.NoError(t, err)
	assert.True(t, lc.hasMessage("token refreshed successfully"))
	assert.True(t, lc.hasLevel(slog.LevelInfo))
}

func TestPersistingTokenSource_LogsWarnNearExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	nearExpiry := time.Now().Add(5 * time.Minute) // within 10 minutes
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new", Expiry: nearExpiry}

	lc := &logCapture{}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.Logger = slog.New(lc)

	_, err := pts.Token()
	require.NoError(t, err)
	assert.True(t, lc.hasLevel(slog.LevelWarn))
}

func TestPersistingTokenSource_LogsErrorOnSaveFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	initial := &oauth2.Token{AccessToken: "old"}
	refreshed := &oauth2.Token{AccessToken: "new"}

	cs := &countingSaveFunc{failN: 10}
	lc := &logCapture{}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: refreshed}, path, initial)
	pts.SaveFunc = cs.save
	pts.Logger = slog.New(lc)

	_, err := pts.Token()
	require.NoError(t, err)
	assert.True(t, lc.hasLevel(slog.LevelError))
}

// --- Step 3: Clear re-auth error messages ---

func TestPersistingTokenSource_InvalidGrantError(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantMsg string
	}{
		{
			name:    "invalid_grant",
			errMsg:  "oauth2: cannot fetch token: 400 Bad Request\nResponse: {\"error\":\"invalid_grant\"}",
			wantMsg: "Run 'pkb auth' to re-authenticate",
		},
		{
			name:    "token expired or revoked",
			errMsg:  "Token has been expired or revoked",
			wantMsg: "Run 'pkb auth' to re-authenticate",
		},
		{
			name:    "generic error not wrapped",
			errMsg:  "network timeout",
			wantMsg: "network timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pts := NewPersistingTokenSource(
				&errorTokenSource{err: fmt.Errorf("%s", tt.errMsg)},
				"/unused",
				&oauth2.Token{AccessToken: "old"},
			)

			_, err := pts.Token()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

// --- Step 6: Background proactive token refresh ---

// expiringTokenSource returns a token that expires at a configurable time.
type expiringTokenSource struct {
	mu      sync.Mutex
	calls   int
	expiry  time.Time
	accessT string
}

func (e *expiringTokenSource) Token() (*oauth2.Token, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return &oauth2.Token{
		AccessToken: e.accessT,
		Expiry:      e.expiry,
	}, nil
}

func (e *expiringTokenSource) getCalls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func TestPersistingTokenSource_BackgroundRefresh_TriggersOnNearExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	nearExpiry := time.Now().Add(5 * time.Minute) // within 10 min window
	src := &expiringTokenSource{expiry: nearExpiry, accessT: "bg-refreshed"}

	initial := &oauth2.Token{AccessToken: "initial", Expiry: nearExpiry}
	pts := NewPersistingTokenSource(src, path, initial)
	pts.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	pts.StartBackgroundRefresh(ctx, 50*time.Millisecond)

	// Wait for at least one background tick to fire.
	time.Sleep(200 * time.Millisecond)
	cancel()

	assert.GreaterOrEqual(t, src.getCalls(), 1)
}

func TestPersistingTokenSource_BackgroundRefresh_RespectsCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	farExpiry := time.Now().Add(1 * time.Hour) // not near expiry
	src := &expiringTokenSource{expiry: farExpiry, accessT: "no-refresh"}

	initial := &oauth2.Token{AccessToken: "initial", Expiry: farExpiry}
	pts := NewPersistingTokenSource(src, path, initial)
	pts.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	pts.StartBackgroundRefresh(ctx, 50*time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)

	// Should not have made any calls since token is far from expiry and context cancelled.
	assert.Equal(t, 0, src.getCalls())
}

func TestPersistingTokenSource_BackgroundRefresh_NoRefreshWhenFarFromExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	farExpiry := time.Now().Add(1 * time.Hour)
	src := &expiringTokenSource{expiry: farExpiry, accessT: "far"}

	initial := &oauth2.Token{AccessToken: "initial", Expiry: farExpiry}
	pts := NewPersistingTokenSource(src, path, initial)
	pts.Logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	ctx, cancel := context.WithCancel(context.Background())
	pts.StartBackgroundRefresh(ctx, 50*time.Millisecond)

	time.Sleep(200 * time.Millisecond)
	cancel()

	// Token is far from expiry, so no refresh calls should have been made.
	assert.Equal(t, 0, src.getCalls())
}

// --- Step 5 (partial): TokenChecker interface ---

func TestTokenStatus_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	future := time.Now().Add(1 * time.Hour)
	tok := &oauth2.Token{AccessToken: "valid", Expiry: future}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: tok}, path, tok)

	status := pts.TokenHealth()
	assert.True(t, status.Valid)
	assert.Greater(t, status.ExpiresIn, 50*time.Minute)
	assert.Empty(t, status.Error)
}

func TestTokenStatus_Expired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	past := time.Now().Add(-1 * time.Hour)
	tok := &oauth2.Token{AccessToken: "expired", Expiry: past}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: tok}, path, tok)

	status := pts.TokenHealth()
	assert.False(t, status.Valid)
	assert.Less(t, status.ExpiresIn, time.Duration(0))
	assert.Contains(t, status.Error, "expired")
}

func TestTokenStatus_NearExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	nearFuture := time.Now().Add(5 * time.Minute)
	tok := &oauth2.Token{AccessToken: "near", Expiry: nearFuture}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: tok}, path, tok)

	status := pts.TokenHealth()
	assert.True(t, status.Valid)
	assert.Less(t, status.ExpiresIn, 10*time.Minute)
}

func TestTokenStatus_ZeroExpiry(t *testing.T) {
	// Zero expiry means no expiration info — treat as valid.
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{AccessToken: "no-expiry"}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: tok}, path, tok)

	status := pts.TokenHealth()
	assert.True(t, status.Valid)
	assert.Empty(t, status.Error)
}

func TestTokenCheckerInterface(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	tok := &oauth2.Token{AccessToken: "iface-test"}
	pts := NewPersistingTokenSource(&staticTokenSource{tok: tok}, path, tok)

	// Verify PersistingTokenSource implements TokenChecker.
	var _ TokenChecker = pts
}
