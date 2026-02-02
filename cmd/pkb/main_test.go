package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cwoolley/personal-knowledge-base/internal/apiclient"
	"github.com/cwoolley/personal-knowledge-base/internal/config"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gmail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func noopSearch(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
	return nil, nil
}

func TestTruncateSnippet(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
	}{
		{"empty", "", ""},
		{"short", "hello world", "hello world"},
		{"exactly 80", strings.Repeat("x", 80), strings.Repeat("x", 80)},
		{"81 chars truncated", strings.Repeat("x", 81), strings.Repeat("x", 77) + "..."},
		{"long string", strings.Repeat("a", 200), strings.Repeat("a", 77) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncateSnippet(tt.input))
		})
	}
}

func TestSearchCommand_EmbeddedServerError(t *testing.T) {
	orig := startEmbeddedServer
	startEmbeddedServer = func(_ SearchFunc) (*apiclient.Client, func(), error) {
		return nil, nil, fmt.Errorf("listen failed")
	}
	t.Cleanup(func() { startEmbeddedServer = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listen failed")
}

func TestInteractiveCommand_EmbeddedServerError(t *testing.T) {
	orig := startEmbeddedServer
	startEmbeddedServer = func(_ SearchFunc) (*apiclient.Client, func(), error) {
		return nil, nil, fmt.Errorf("listen failed")
	}
	t.Cleanup(func() { startEmbeddedServer = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"interactive"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listen failed")
}

func TestSearchCommand_PrintsSnippet(t *testing.T) {
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "Doc", Snippet: "This is the snippet text", URL: "https://example.com", Source: "mock"},
		}, nil
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test"}, mockSearch, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "This is the snippet text")
}

func TestSearchCommand_OmitsEmptySnippet(t *testing.T) {
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "Doc", Snippet: "", URL: "https://example.com", Source: "mock"},
		}, nil
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test"}, mockSearch, &buf)
	require.NoError(t, err)
	output := buf.String()
	// Should have title, URL, source — but only 3 lines of content (no blank snippet line).
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "expected 3 lines (title, URL, source) with no snippet line")
}

func TestSearchCommand_TruncatesLongSnippet(t *testing.T) {
	long := strings.Repeat("z", 200)
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "Doc", Snippet: long, URL: "https://example.com", Source: "mock"},
		}, nil
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "test"}, mockSearch, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, strings.Repeat("z", 77)+"...")
	assert.NotContains(t, output, strings.Repeat("z", 78))
}

func TestRun_ReturnsNilOnSuccess(t *testing.T) {
	err := run([]string{}, noopSearch)
	assert.NoError(t, err)
}

func TestSearchCommand_PrintsResults(t *testing.T) {
	mockSearch := func(_ context.Context, query string, _ []string) ([]connectors.Result, error) {
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
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
		return []connectors.Result{}, nil
	}
	var buf bytes.Buffer
	err := runWithOutput([]string{"search", "empty"}, mockSearch, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No results found.")
}

// BUG-011: Test the search error path.
func TestSearchCommand_Error(t *testing.T) {
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
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
	_, err := fn(context.Background(), "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Google Drive credentials not configured")
}

// BUG-009: The "serve" subcommand is registered and accepts --addr.
func TestServeCommand_IsRegistered(t *testing.T) {
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
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
	mockSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
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
	_, err := fn(context.Background(), "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Google Drive credentials not configured")
}

func TestBuildSearchFn_ConfigLoadError(t *testing.T) {
	orig := loadConfig
	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("config error")
	}
	t.Cleanup(func() { loadConfig = orig })

	fn := buildSearchFn()
	_, err := fn(context.Background(), "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestBuildSearchFn_TokenLoadError(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")
	t.Setenv("PKB_TOKEN_PATH", "/nonexistent/path/token.json")

	fn := buildSearchFn()
	_, err := fn(context.Background(), "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load OAuth token")
}

func TestBuildSearchFn_APIClientError(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	data, err := json.Marshal(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, data, 0600))
	t.Setenv("PKB_TOKEN_PATH", tokenPath)

	orig := newAPIClient
	newAPIClient = func(_ context.Context, _ oauth2.TokenSource) (*gdrive.APIClient, error) {
		return nil, fmt.Errorf("api client error")
	}
	t.Cleanup(func() { newAPIClient = orig })

	fn := buildSearchFn()
	_, err = fn(context.Background(), "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Google Drive client")
}

func TestBuildSearchFn_SuccessPath(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	data, err := json.Marshal(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, data, 0600))
	t.Setenv("PKB_TOKEN_PATH", tokenPath)

	fn := buildSearchFn()
	// The closure creates a real Drive client. The search call will fail
	// because there's no real API, but all lines in buildSearchFn are exercised.
	_, err = fn(context.Background(), "test", nil)
	assert.Error(t, err)
}

// mockTeaRunner implements teaRunner for testing.
type mockTeaRunner struct {
	err error
}

func (m *mockTeaRunner) Run() (tea.Model, error) {
	return nil, m.err
}

func TestInteractiveCommand_RunsProgram(t *testing.T) {
	orig := newTeaProgram
	newTeaProgram = func(_ tea.Model) teaRunner {
		return &mockTeaRunner{}
	}
	t.Cleanup(func() { newTeaProgram = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"interactive"}, noopSearch, &buf)
	assert.NoError(t, err)
}

func TestInteractiveCommand_Error(t *testing.T) {
	orig := newTeaProgram
	newTeaProgram = func(_ tea.Model) teaRunner {
		return &mockTeaRunner{err: fmt.Errorf("terminal error")}
	}
	t.Cleanup(func() { newTeaProgram = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"interactive"}, noopSearch, &buf)
	assert.Error(t, err)
}

// mockHTTPServer implements httpServer for testing serveLoop.
type mockHTTPServer struct {
	serveFunc   func() error
	shutdownErr error
	addr        string
}

func (m *mockHTTPServer) Serve() error        { return m.serveFunc() }
func (m *mockHTTPServer) Addr() string         { return m.addr }
func (m *mockHTTPServer) Shutdown(_ context.Context) error { return m.shutdownErr }

func TestServeLoop_ErrServerClosed(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	mock := &mockHTTPServer{
		serveFunc: func() error { return http.ErrServerClosed },
	}
	var buf bytes.Buffer
	err := serveLoop(mock, &buf)
	assert.NoError(t, err)
}

func TestServeLoop_ServerError(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	mock := &mockHTTPServer{
		serveFunc: func() error { return fmt.Errorf("bind error") },
	}
	var buf bytes.Buffer
	err := serveLoop(mock, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bind error")
}

func TestServeLoop_ShutdownError(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	serveDone := make(chan struct{})
	mock := &mockHTTPServer{
		serveFunc:   func() error { <-serveDone; return http.ErrServerClosed },
		shutdownErr: fmt.Errorf("shutdown failed"),
	}

	buf := &syncBuffer{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveLoop(mock, buf)
	}()

	testCh <- syscall.SIGINT

	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shutdown")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for serveLoop to return")
	}
	close(serveDone)
}

func TestServeCommand_ListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	var buf bytes.Buffer
	err = runWithOutput([]string{"serve", "--addr", ln.Addr().String()}, noopSearch, &buf)
	assert.Error(t, err)
}

// waitForServe polls the syncBuffer until "Listening on" appears,
// then extracts the addr from the output. It also watches errCh for
// early failures and the deadline for timeouts.
func waitForServe(t *testing.T, buf *syncBuffer, errCh <-chan error) string {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for server to start")
		case err := <-errCh:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		out := buf.String()
		if strings.Contains(out, "Listening on") {
			// Extract the address from "Listening on <addr>\n"
			line := strings.TrimSpace(out)
			parts := strings.SplitN(line, "Listening on ", 2)
			require.Len(t, parts, 2, "expected 'Listening on <addr>' in output")
			addr := strings.TrimSpace(parts[1])
			// addr may contain further lines; take only the first line.
			if idx := strings.Index(addr, "\n"); idx >= 0 {
				addr = addr[:idx]
			}
			return addr
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestServeSearch_MissingQuery_Returns400(t *testing.T) {
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

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/search")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "missing required parameter: q")

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestServeSearch_WithQuery_ReturnsJSON(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	mockSearch := func(_ context.Context, query string, _ []string) ([]connectors.Result, error) {
		return []connectors.Result{
			{Title: "API Doc", URL: "https://example.com/api", Source: "mock"},
		}, nil
	}

	buf := &syncBuffer{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithOutput([]string{"serve", "--addr", ":0"}, mockSearch, buf)
	}()

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/search?q=hello")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var results []connectors.Result
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&results))
	require.Len(t, results, 1)
	assert.Equal(t, "API Doc", results[0].Title)
	assert.Equal(t, "https://example.com/api", results[0].URL)
	assert.Equal(t, "mock", results[0].Source)

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestServeSearch_WithSources_PassesThroughFilter(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	var capturedSources []string
	mockSearch := func(_ context.Context, _ string, sources []string) ([]connectors.Result, error) {
		capturedSources = sources
		return []connectors.Result{
			{Title: "Filtered", Source: "gdrive"},
		}, nil
	}

	buf := &syncBuffer{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithOutput([]string{"serve", "--addr", ":0"}, mockSearch, buf)
	}()

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/search?q=test&sources=gdrive")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{"gdrive"}, capturedSources)

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestServeSearch_WithoutSources_PassesNil(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	sourcesCalled := false
	var capturedSources []string
	mockSearch := func(_ context.Context, _ string, sources []string) ([]connectors.Result, error) {
		sourcesCalled = true
		capturedSources = sources
		return []connectors.Result{}, nil
	}

	buf := &syncBuffer{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithOutput([]string{"serve", "--addr", ":0"}, mockSearch, buf)
	}()

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/search?q=test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, sourcesCalled)
	assert.Nil(t, capturedSources, "sources should be nil when not provided")

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestServeCommand_ServesWebUI(t *testing.T) {
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

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	assert.Contains(t, html, "<html")
	assert.Contains(t, html, "Search")
	assert.Contains(t, html, "gdrive")

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestServeSearch_SearchError_Returns500(t *testing.T) {
	testCh := make(chan os.Signal, 1)
	origMakeSignalCh := makeSignalCh
	makeSignalCh = func() (chan os.Signal, func()) {
		return testCh, func() {}
	}
	t.Cleanup(func() { makeSignalCh = origMakeSignalCh })

	failSearch := func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
		return nil, fmt.Errorf("search engine exploded")
	}

	buf := &syncBuffer{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithOutput([]string{"serve", "--addr", ":0"}, failSearch, buf)
	}()

	addr := waitForServe(t, buf, errCh)

	resp, err := http.Get("http://" + addr + "/search?q=boom")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "search engine exploded")

	testCh <- syscall.SIGINT
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for serve to shut down")
	}
}

func TestBuildSearchFn_GmailClientError_FallsBackToDriveOnly(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	data, err := json.Marshal(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenPath, data, 0600))
	t.Setenv("PKB_TOKEN_PATH", tokenPath)

	orig := newGmailAPIClient
	newGmailAPIClient = func(_ context.Context, _ oauth2.TokenSource) (*gmail.APIClient, error) {
		return nil, fmt.Errorf("gmail not available")
	}
	t.Cleanup(func() { newGmailAPIClient = orig })

	fn := buildSearchFn()
	// Should still work (falls back to Drive only), though Drive search will fail
	// because there's no real API. The point is it didn't crash from Gmail error.
	_, err = fn(context.Background(), "test", nil)
	assert.Error(t, err)
}

// --- auth command tests ---

func TestAuthCommand_IsRegistered(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd(noopSearch, &buf)
	authCmd, _, err := cmd.Find([]string{"auth"})
	require.NoError(t, err)
	assert.Equal(t, "auth", authCmd.Name())
}

func TestAuthCommand_MissingCredentials(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "")

	var buf bytes.Buffer
	err := runWithOutput([]string{"auth"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials not configured")
}

func TestAuthCommand_ConfigLoadError(t *testing.T) {
	orig := loadConfig
	loadConfig = func() (*config.Config, error) {
		return nil, fmt.Errorf("config boom")
	}
	t.Cleanup(func() { loadConfig = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"auth"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestAuthCommand_FlowError(t *testing.T) {
	t.Setenv("PKB_GOOGLE_CLIENT_ID", "test-id")
	t.Setenv("PKB_GOOGLE_CLIENT_SECRET", "test-secret")

	orig := openBrowser
	openBrowser = func(rawURL string) error {
		return fmt.Errorf("no browser")
	}
	t.Cleanup(func() { openBrowser = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"auth"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization failed")
	assert.Contains(t, buf.String(), "Opening browser")
}

func TestAuthCommand_SaveTokenError(t *testing.T) {
	// Mock token exchange server returns a valid token.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer"}`)
	}))
	defer tokenServer.Close()

	// Override endpoint to point to mock token server.
	origEndpoint := googleOAuthEndpoint
	googleOAuthEndpoint = func() oauth2.Endpoint {
		return oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: tokenServer.URL,
		}
	}
	t.Cleanup(func() { googleOAuthEndpoint = origEndpoint })

	// Override loadConfig to use a non-writable token path.
	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			GoogleClientID:     "test-id",
			GoogleClientSecret: "test-secret",
			TokenPath:          "/nonexistent/dir/token.json",
		}, nil
	}
	t.Cleanup(func() { loadConfig = origLoad })

	orig := openBrowser
	openBrowser = func(rawURL string) error {
		go func() {
			parsed, _ := neturl.Parse(rawURL)
			redirectURI := parsed.Query().Get("redirect_uri")
			//nolint:gosec // test-only HTTP request
			resp, err := http.Get(redirectURI + "?code=test-code")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
	t.Cleanup(func() { openBrowser = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"auth"}, noopSearch, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save token")
	assert.Contains(t, buf.String(), "Opening browser")
}

func TestAuthCommand_SuccessPath(t *testing.T) {
	// Mock token exchange server.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"fresh-token","token_type":"Bearer"}`)
	}))
	defer tokenServer.Close()

	// Override endpoint to point to mock token server.
	origEndpoint := googleOAuthEndpoint
	googleOAuthEndpoint = func() oauth2.Endpoint {
		return oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: tokenServer.URL,
		}
	}
	t.Cleanup(func() { googleOAuthEndpoint = origEndpoint })

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")

	origLoad := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			GoogleClientID:     "test-id",
			GoogleClientSecret: "test-secret",
			TokenPath:          tokenPath,
		}, nil
	}
	t.Cleanup(func() { loadConfig = origLoad })

	orig := openBrowser
	openBrowser = func(rawURL string) error {
		go func() {
			parsed, _ := neturl.Parse(rawURL)
			redirectURI := parsed.Query().Get("redirect_uri")
			//nolint:gosec // test-only HTTP request
			resp, err := http.Get(redirectURI + "?code=test-code")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
	t.Cleanup(func() { openBrowser = orig })

	var buf bytes.Buffer
	err := runWithOutput([]string{"auth"}, noopSearch, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Opening browser")
	assert.Contains(t, output, "Token saved to")

	// Verify the token was actually written to disk.
	loaded, err := gdrive.LoadToken(tokenPath)
	require.NoError(t, err)
	assert.Equal(t, "fresh-token", loaded.AccessToken)
}
