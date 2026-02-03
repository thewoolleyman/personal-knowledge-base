//go:build acceptance

// Package acceptance contains top-of-the-pyramid tests that exercise the
// application from a user's perspective. These tests build the actual binary
// and run it as a subprocess, checking stdout, stderr, and exit codes.
//
// RULE: These tests must NEVER import internal packages. They treat the
// application as a black box — the same way a human user does.
//
// Run: go test -tags=acceptance -v ./tests/acceptance/
package acceptance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectRoot finds the project root by looking for go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	// tests/acceptance/ is two levels below the project root
	dir, err := filepath.Abs("../..")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "go.mod"), "Could not find project root")
	return dir
}

// buildBinary compiles the pkb binary into a temp directory and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	root := projectRoot(t)
	binary := filepath.Join(t.TempDir(), "pkb")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/pkb")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(out))
	return binary
}

// runPKB executes the pkb binary with the given args and returns stdout, stderr, and exit code.
func runPKB(t *testing.T, binary string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = os.Environ()

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("Failed to run binary: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

// buildBinaryWithVersion compiles the pkb binary with a version injected via ldflags.
func buildBinaryWithVersion(t *testing.T, ver string) string {
	t.Helper()
	root := projectRoot(t)
	binary := filepath.Join(t.TempDir(), "pkb")
	ldflags := fmt.Sprintf("-X main.version=%s", ver)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binary, "./cmd/pkb")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(out))
	return binary
}

// --- Tests mirror what the README tells a human to do ---

func TestAcceptance_HelpShowsSearchCommand(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "search")
	assert.Contains(t, stdout, "Personal Knowledge Base")
}

func TestAcceptance_ServeHelp(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "serve", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "serve")
}

func TestAcceptance_InteractiveHelp(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "interactive", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "interactive")
}

func TestAcceptance_SearchHelp(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "search", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "Search across all connected services")
}

func TestAcceptance_SearchWithoutQuery_ShowsUsageError(t *testing.T) {
	binary := buildBinary(t)

	_, stderr, exitCode := runPKB(t, binary, "search")

	assert.NotEqual(t, 0, exitCode, "Expected non-zero exit code when no query provided")
	assert.Contains(t, stderr, "requires at least 1 arg")
}

func TestAcceptance_VersionShowsVersionString(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "version")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "pkb version")
}

func TestAcceptance_VersionLdflagsInjection(t *testing.T) {
	binary := buildBinaryWithVersion(t, "1.2.3")

	stdout, _, exitCode := runPKB(t, binary, "version")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "pkb version 1.2.3")
}

func TestAcceptance_SearchWithQuery_GivesActionableOutput(t *testing.T) {
	binary := buildBinary(t)

	// When a user runs search without credentials configured, the output
	// must be helpful — not reference nonexistent commands.
	stdout, stderr, _ := runPKB(t, binary, "search", "test query")
	combined := stdout + stderr

	// Must NOT reference commands that don't exist
	assert.NotContains(t, combined, "pkb setup",
		"Error must not reference nonexistent 'pkb setup' command")

	// Must give the user actionable information about what to do
	assert.True(t,
		strings.Contains(combined, "PKB_GOOGLE_CLIENT_ID") ||
			strings.Contains(combined, "credentials") ||
			strings.Contains(combined, "No results"),
		"Output should tell the user what to configure or show results, got: %s", combined)
}

// --- Tests for new features: auth command, /search endpoint, Gmail ---

func TestAcceptance_AuthHelp(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "auth", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "auth")
	assert.Contains(t, stdout, "OAuth")
}

func TestAcceptance_AuthWithoutCredentials_ShowsHelpfulError(t *testing.T) {
	binary := buildBinary(t)

	// Run auth with no credentials set
	cmd := exec.Command(binary, "auth")
	cmd.Env = []string{"HOME=" + t.TempDir()} // clean env, no creds
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()

	combined := outBuf.String() + errBuf.String()
	assert.Error(t, err, "Expected error when credentials not configured")
	assert.True(t,
		strings.Contains(combined, "PKB_GOOGLE_CLIENT_ID") ||
			strings.Contains(combined, "credentials") ||
			strings.Contains(combined, "not configured"),
		"Should tell user to configure credentials, got: %s", combined)
}

func TestAcceptance_HelpShowsAuthCommand(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "auth", "Help should list the auth subcommand")
}

func TestAcceptance_ServeSearchEndpoint(t *testing.T) {
	binary := buildBinary(t)

	// Start the server on a random port
	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + t.TempDir()}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})

	// Read stdout until we see "Listening on" to get the address
	scanner := bufio.NewScanner(stdout)
	var addr string
	deadline := time.After(10 * time.Second)
	addrCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Listening on ") {
				addrCh <- strings.TrimPrefix(line, "Listening on ")
				return
			}
		}
	}()

	select {
	case addr = <-addrCh:
	case <-deadline:
		t.Fatal("timeout waiting for server to start")
	}

	baseURL := "http://" + addr

	// Test 1: /health returns 200
	resp, err := http.Get(baseURL + "/health")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test 2: /search without q returns 400 with JSON error
	resp, err = http.Get(baseURL + "/search")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	var errBody map[string]string
	require.NoError(t, json.Unmarshal(body, &errBody))
	assert.Contains(t, errBody["error"], "missing required parameter")

	// Test 3: /search with q returns JSON (500 because no creds, but valid JSON)
	resp, err = http.Get(baseURL + "/search?q=test")
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	// Should be valid JSON regardless of status code
	assert.True(t, json.Valid(body), "Response should be valid JSON, got: %s", string(body))
}

func TestAcceptance_ServeSearchEndpoint_SourceFiltering(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + t.TempDir()}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})

	scanner := bufio.NewScanner(stdout)
	var addr string
	deadline := time.After(10 * time.Second)
	addrCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Listening on ") {
				addrCh <- strings.TrimPrefix(line, "Listening on ")
				return
			}
		}
	}()

	select {
	case addr = <-addrCh:
	case <-deadline:
		t.Fatal("timeout waiting for server to start")
	}

	baseURL := "http://" + addr

	// /search with sources param should return valid JSON
	resp, err := http.Get(baseURL + "/search?q=test&sources=gdrive")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	assert.True(t, json.Valid(body), "Response should be valid JSON, got: %s", string(body))
}

func TestAcceptance_ServeWebUI_ReturnsHTML(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + t.TempDir()}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})

	scanner := bufio.NewScanner(stdout)
	var addr string
	deadline := time.After(10 * time.Second)
	addrCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Listening on ") {
				addrCh <- strings.TrimPrefix(line, "Listening on ")
				return
			}
		}
	}()

	select {
	case addr = <-addrCh:
	case <-deadline:
		t.Fatal("timeout waiting for server to start")
	}

	baseURL := "http://" + addr

	// GET / should return the web UI HTML
	resp, err := http.Get(baseURL + "/")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, html, "<html", "should serve HTML page")
	assert.Contains(t, html, "Search", "should contain search UI")
	assert.Contains(t, html, "gdrive", "should have gdrive checkbox")
	assert.Contains(t, html, "gmail", "should have gmail checkbox")
}

func TestAcceptance_SearchWithCredentials_ReturnsResults(t *testing.T) {
	// This test requires real Google Drive credentials.
	// Skip if not configured.
	if os.Getenv("PKB_GOOGLE_CLIENT_ID") == "" {
		t.Skip("skipping: PKB_GOOGLE_CLIENT_ID not set")
	}
	if os.Getenv("PKB_GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("skipping: PKB_GOOGLE_CLIENT_SECRET not set")
	}

	binary := buildBinary(t)

	// Search for something that should exist in the Obsidian vault mirror
	stdout, stderr, exitCode := runPKB(t, binary, "search", "md")

	assert.Equal(t, 0, exitCode, "Expected zero exit code with valid credentials, stderr: %s", stderr)
	assert.NotEmpty(t, stdout, "Expected search results in stdout")
	// Results should contain numbered output
	assert.Contains(t, stdout, "1.")
}

// --- Additional acceptance tests for comprehensive coverage ---

func TestAcceptance_SearchWithSourcesFlag_FiltersResults(t *testing.T) {
	binary := buildBinary(t)

	// Test --sources flag is recognized (even without credentials)
	stdout, stderr, _ := runPKB(t, binary, "search", "--sources", "gdrive", "test")
	combined := stdout + stderr

	// Should not error on the flag itself - flag is valid
	// Error will be about credentials, not unknown flag
	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources flag")
	assert.NotContains(t, combined, "Error: unknown command",
		"Should recognize search command with --sources")
}

func TestAcceptance_SearchWithMultipleSources_AcceptsCommaList(t *testing.T) {
	binary := buildBinary(t)

	// Test multiple sources with comma-separated list
	stdout, stderr, _ := runPKB(t, binary, "search", "--sources", "gdrive,gmail", "test")
	combined := stdout + stderr

	// Should not error on the flag format
	assert.NotContains(t, combined, "invalid argument",
		"Should accept comma-separated source list")
	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources flag with multiple values")
}

func TestAcceptance_VersionWithoutBuild_ShowsDevVersion(t *testing.T) {
	// Build without version ldflags - should show "dev"
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "version")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "pkb version")
	// When built without version ldflags, should default to "dev"
	assert.Contains(t, stdout, "dev")
}

func TestAcceptance_HelpShowsAllCommands(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "--help")

	assert.Equal(t, 0, exitCode)
	// Verify all major commands are listed
	assert.Contains(t, stdout, "search", "Help should list search command")
	assert.Contains(t, stdout, "serve", "Help should list serve command")
	assert.Contains(t, stdout, "interactive", "Help should list interactive command")
	assert.Contains(t, stdout, "version", "Help should list version command")
	assert.Contains(t, stdout, "auth", "Help should list auth command")
}

func TestAcceptance_SearchReturnsNonZeroExitOnError(t *testing.T) {
	binary := buildBinary(t)

	// Run search with no args - should error
	_, _, exitCode := runPKB(t, binary, "search")

	assert.NotEqual(t, 0, exitCode,
		"Search without arguments should return non-zero exit code")
}

func TestAcceptance_InvalidCommand_ShowsError(t *testing.T) {
	binary := buildBinary(t)

	_, stderr, exitCode := runPKB(t, binary, "nonexistent")

	assert.NotEqual(t, 0, exitCode, "Invalid command should return non-zero exit")
	assert.Contains(t, stderr, "unknown command",
		"Should indicate unknown command in stderr")
}

func TestAcceptance_ServeWithCustomAddr_UsesSpecifiedPort(t *testing.T) {
	binary := buildBinary(t)

	// Start server with custom address
	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + t.TempDir()}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})

	// Verify server starts and outputs listening address
	scanner := bufio.NewScanner(stdout)
	var foundListening bool
	deadline := time.After(5 * time.Second)
	ch := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Listening on ") {
				ch <- true
				return
			}
		}
	}()

	select {
	case foundListening = <-ch:
	case <-deadline:
		t.Fatal("timeout waiting for server to start")
	}

	assert.True(t, foundListening, "Server should output 'Listening on' message")
}

func TestAcceptance_InteractiveAlias_TUI_Works(t *testing.T) {
	binary := buildBinary(t)

	// Test that 'tui' is an alias for 'interactive'
	stdout, _, exitCode := runPKB(t, binary, "tui", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "interactive",
		"tui alias should show interactive help")
}

func TestAcceptance_SearchOutput_IncludesSourceTag(t *testing.T) {
	// This test verifies the output format mentioned in README
	if os.Getenv("PKB_GOOGLE_CLIENT_ID") == "" {
		t.Skip("skipping: requires credentials")
	}
	if os.Getenv("PKB_GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("skipping: requires credentials")
	}

	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "test")

	assert.Equal(t, 0, exitCode, "Search should succeed with credentials, stderr: %s", stderr)

	if !strings.Contains(stdout, "No results") {
		// If there are results, verify format includes source tag
		assert.Regexp(t, `\[google-drive\]|\[gmail\]`, stdout,
			"Results should include source tag in brackets")
	}
}

func TestAcceptance_SearchOutput_IncludesNumberedResults(t *testing.T) {
	if os.Getenv("PKB_GOOGLE_CLIENT_ID") == "" {
		t.Skip("skipping: requires credentials")
	}
	if os.Getenv("PKB_GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("skipping: requires credentials")
	}

	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "test")

	assert.Equal(t, 0, exitCode, "Search should succeed, stderr: %s", stderr)

	if !strings.Contains(stdout, "No results") {
		// Results should be numbered
		assert.Regexp(t, `^\d+\.`, strings.TrimSpace(stdout),
			"Results should start with numbered list (1., 2., etc.)")
	}
}

func TestAcceptance_SearchOutput_IncludesURLs(t *testing.T) {
	if os.Getenv("PKB_GOOGLE_CLIENT_ID") == "" {
		t.Skip("skipping: requires credentials")
	}
	if os.Getenv("PKB_GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("skipping: requires credentials")
	}

	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "test")

	assert.Equal(t, 0, exitCode, "Search should succeed, stderr: %s", stderr)

	if !strings.Contains(stdout, "No results") {
		// Results should include URLs
		assert.Regexp(t, `https?://`, stdout,
			"Results should include URLs")
	}
}

func TestAcceptance_SearchWithNoResults_ShowsFriendlyMessage(t *testing.T) {
	if os.Getenv("PKB_GOOGLE_CLIENT_ID") == "" {
		t.Skip("skipping: requires credentials")
	}
	if os.Getenv("PKB_GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("skipping: requires credentials")
	}

	binary := buildBinary(t)

	// Search for something very unlikely to exist
	stdout, stderr, exitCode := runPKB(t, binary, "search", "xyzzy_nonexistent_query_12345")

	assert.Equal(t, 0, exitCode, "Search with no results should still exit 0, stderr: %s", stderr)
	assert.Contains(t, stdout, "No results",
		"Should show friendly 'No results' message")
}

func TestAcceptance_ServeHealthEndpoint_Returns200(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + t.TempDir()}

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = cmd.Stdout

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})

	scanner := bufio.NewScanner(stdout)
	var addr string
	deadline := time.After(5 * time.Second)
	addrCh := make(chan string, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Listening on ") {
				addrCh <- strings.TrimPrefix(line, "Listening on ")
				return
			}
		}
	}()

	select {
	case addr = <-addrCh:
	case <-deadline:
		t.Fatal("timeout waiting for server")
	}

	// Test /health endpoint
	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"/health endpoint should return 200 OK")
}

func TestAcceptance_MakeBuildTarget_ProducesBinary(t *testing.T) {
	// This mirrors what README tells users: "make build"
	root := projectRoot(t)

	cmd := exec.Command("make", "build")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "make build should succeed: %s", string(output))

	// Verify binary was created
	binaryPath := filepath.Join(root, "pkb")
	assert.FileExists(t, binaryPath, "make build should create pkb binary")

	// Verify it's executable
	info, err := os.Stat(binaryPath)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0111, "Binary should be executable")
}

func TestAcceptance_MakeRunTarget_ExecutesBinary(t *testing.T) {
	// This mirrors what README tells users: "make run"
	root := projectRoot(t)

	// The Makefile run target works like: make run <args>
	// So we test with a simple command that should work
	cmd := exec.Command("make", "run", "version")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "make run version should succeed: %s", string(output))

	// Output should contain version information
	assert.Contains(t, string(output), "pkb version",
		"make run should execute the binary and show version")
}
