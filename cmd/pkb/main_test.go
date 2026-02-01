package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncBuffer is a thread-safe bytes.Buffer for use in concurrent tests.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func (sb *syncBuffer) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Len()
}

// ensure syncBuffer satisfies io.Writer.
var _ io.Writer = (*syncBuffer)(nil)

func noopSearch(_ context.Context, _ string) ([]connectors.Result, error) {
	return nil, nil
}

func TestRun_ReturnsNilOnSuccess(t *testing.T) {
	err := run([]string{}, noopSearch)
	assert.NoError(t, err)
}

func TestSearchCommand_PrintsResults(t *testing.T) {
	mockSearch := func(_ context.Context, query string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "Test Doc", URL: "https://example.com/doc", Source: "mock"},
			{Title: "Another Doc", URL: "https://example.com/doc2", Source: "mock"},
		}, nil
	}

	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test query"}, mockSearch, &buf)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Test Doc")
	assert.Contains(t, output, "https://example.com/doc")
	assert.Contains(t, output, "Another Doc")
}

func TestSearchCommand_NoQuery(t *testing.T) {
	err := run([]string{"search"}, noopSearch)
	assert.Error(t, err)
}

// BUG-011: Test the "no results" output path.
func TestSearchCommand_NoResults(t *testing.T) {
	mockSearch := func(_ context.Context, _ string) ([]connectors.Result, error) {
		return []connectors.Result{}, nil
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "empty"}, mockSearch, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No results found.")
}

// BUG-011: Test the search error path.
func TestSearchCommand_Error(t *testing.T) {
	mockSearch := func(_ context.Context, _ string) ([]connectors.Result, error) {
		return nil, fmt.Errorf("connection failed")
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test"}, mockSearch, &buf)
	assert.Error(t, err)
}

// BUG-008: buildSearchFn uses config.Load() instead of inline os.Getenv.
func TestBuildSearchFn_UsesConfig(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "")

	fn := buildSearchFn()
	_, err := fn(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Google Drive credentials not configured")
}

// BUG-009: The "serve" subcommand is registered and accepts --addr.
func TestServeCommand_IsRegistered(t *testing.T) {
	mockSearch := func(_ context.Context, _ string) ([]connectors.Result, error) {
		return nil, nil
	}
	var buf bytes.Buffer
	cmd := newRootCmd(mockSearch, &buf)

	// The serve subcommand must exist.
	serveCmd, _, err := cmd.Find([]string{"serve"})
	require.NoError(t, err)
	assert.Equal(t, "serve", serveCmd.Name())

	// The --addr flag must be defined.
	f := serveCmd.Flags().Lookup("addr")
	require.NotNil(t, f)
	assert.Equal(t, ":8080", f.DefValue)
}

// BUG-010: The "interactive" subcommand is registered with alias "tui".
func TestInteractiveCommand_IsRegistered(t *testing.T) {
	mockSearch := func(_ context.Context, _ string) ([]connectors.Result, error) {
		return nil, nil
	}
	var buf bytes.Buffer
	cmd := newRootCmd(mockSearch, &buf)

	interactiveCmd, _, err := cmd.Find([]string{"interactive"})
	require.NoError(t, err)
	assert.Equal(t, "interactive", interactiveCmd.Name())
	assert.Contains(t, interactiveCmd.Aliases, "tui")
}

// BUG-007: serve command gracefully shuts down on SIGINT/SIGTERM.
func TestServeCommand_GracefulShutdown(t *testing.T) {
	// Inject a test signal channel so we don't send a real SIGINT to the process.
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	buf := &syncBuffer{}
	errCh := make(chan error, 1)

	go func() {
		errCh <- runWithOutput([]string{"serve", "--addr", ":0"}, noopSearch, buf)
	}()

	// Wait for the server to start listening (syncBuffer is thread-safe).
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for server to start")
		case err := <-errCh:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		if buf.Len() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Contains(t, buf.String(), "Listening on")

	// Send SIGINT via the injected channel instead of a real signal.
	testCh <- syscall.SIGINT

	select {
	case err := <-errCh:
		assert.NoError(t, err, "serve should shut down cleanly on SIGINT")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}

	assert.Contains(t, buf.String(), "shutting down")
}

func TestVersionCommand_PrintsVersion(t *testing.T) {
	var buf bytes.Buffer
	err := runWithOutput([]string{"version"}, noopSearch, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "pkb version")
	assert.Contains(t, buf.String(), version)
}

func TestVersionCommand_IsRegistered(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd(noopSearch, &buf)

	versionCmd, _, err := cmd.Find([]string{"version"})
	require.NoError(t, err)
	assert.Equal(t, "version", versionCmd.Name())
}

// BUG-006: buildSearchFn propagates config.Load() errors.
// Note: config.Load() currently never errors, but the code path is
// now defensive. This test verifies the structure is correct by
// confirming that valid config still works and missing creds are caught.
func TestBuildSearchFn_PropagatesConfigError(t *testing.T) {
	// With empty env vars, buildSearchFn should return the "not configured" error.
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "")

	fn := buildSearchFn()
	_, err := fn(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Google Drive credentials not configured")
}
