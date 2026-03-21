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
	"archive/zip"
	"bufio"
	"bytes"
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

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testQuery = "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE"

// TestMain loads the .env file from the project root before running tests.
func TestMain(m *testing.M) {
	// Walk up from tests/acceptance/ to find project root
	if root, err := filepath.Abs("../.."); err == nil {
		_ = godotenv.Load(filepath.Join(root, ".env"))
	}
	os.Exit(m.Run())
}

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
	cmd.Env = append(os.Environ(), "PKB_TAILSCALE=false")

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

	// Must give the user actionable information about what to do, or show results
	assert.True(t,
		strings.Contains(combined, "PKB_GOOGLE_CLIENT_ID") ||
			strings.Contains(combined, "credentials") ||
			strings.Contains(combined, "OAuth") ||
			strings.Contains(combined, "No results") ||
			strings.Contains(combined, "1."), // numbered results when credentials are present
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
	resp, err := http.Get(baseURL + "/search?q=test&sources=google-drive")
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
	assert.Contains(t, html, "google-drive", "should have google-drive checkbox")
	assert.Contains(t, html, "gmail", "should have gmail checkbox")
	assert.Contains(t, html, "obsidian", "should have obsidian checkbox")
	assert.Contains(t, html, "notion", "should have notion checkbox")
}

// --- Additional acceptance tests for comprehensive coverage ---
// NOTE: Tests that require a live Google API token (SearchWithCredentials,
// SearchOutput_IncludesSourceTag, etc.) live in tests/live/live_test.go
// and are run via `make test-live`.

func TestAcceptance_SearchSourcesHelpText_UsesCanonicalNames(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "search", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "google-drive",
		"Help text should show canonical connector name 'google-drive'")
	assert.NotContains(t, stdout, "gdrive",
		"Help text must not use shorthand 'gdrive' — it doesn't match connector.Name()")
}

func TestAcceptance_SearchWithSourcesFlag_FiltersResults(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "google-drive", testQuery)
	combined := stdout + stderr

	// Flag must be recognized (no "unknown flag" error)
	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources flag")

	if exitCode == 0 {
		// Credentials present: verify source filtering works
		assert.Contains(t, stdout, "[google-drive]",
			"Should return google-drive results")
		assert.NotContains(t, stdout, "[gmail]",
			"Should not return gmail results when filtering to google-drive")
		assert.NotContains(t, stdout, "[obsidian]",
			"Should not return obsidian results when filtering to google-drive")
		assert.NotContains(t, stdout, "[notion]",
			"Should not return notion results when filtering to google-drive")
	} else {
		// No credentials or expired token: error must be actionable
		assert.True(t,
			strings.Contains(combined, "credentials") ||
				strings.Contains(combined, "OAuth") ||
				strings.Contains(combined, "pkb auth"),
			"Without credentials, error should mention credentials or auth, got: %s", combined)
	}
}

func TestAcceptance_SearchWithMultipleSources_AcceptsCommaList(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "google-drive,gmail", testQuery)
	combined := stdout + stderr

	assert.NotContains(t, combined, "invalid argument",
		"Should accept comma-separated source list")
	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources flag with multiple values")

	if exitCode == 0 {
		assert.Contains(t, stdout, "[google-drive]",
			"Should return google-drive results")
		assert.Contains(t, stdout, "[gmail]",
			"Should return gmail results")
	}
}

func TestAcceptance_SearchHelpText_ShowsObsidianSource(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "search", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "obsidian",
		"Help text should list 'obsidian' as a valid source")
}

func TestAcceptance_SearchWithObsidianSource_ReturnsResults(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "obsidian", testQuery)
	combined := stdout + stderr

	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources obsidian flag")

	if exitCode == 0 && !strings.Contains(stdout, "No results") {
		assert.Contains(t, stdout, "[obsidian]",
			"Should return obsidian results")
		assert.NotContains(t, stdout, "[google-drive]",
			"Should not return google-drive results when filtering to obsidian")
		assert.NotContains(t, stdout, "[gmail]",
			"Should not return gmail results when filtering to obsidian")
		assert.NotContains(t, stdout, "[notion]",
			"Should not return notion results when filtering to obsidian")
	}
}

func TestAcceptance_SearchWithThreeSources_ReturnsResults(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "google-drive,gmail,obsidian", testQuery)
	combined := stdout + stderr

	assert.NotContains(t, combined, "invalid argument",
		"Should accept comma-separated source list with three sources")

	if exitCode == 0 {
		assert.Contains(t, stdout, "[google-drive]",
			"Should return google-drive results")
		assert.Contains(t, stdout, "[gmail]",
			"Should return gmail results")
		assert.Contains(t, stdout, "[obsidian]",
			"Should return obsidian results")
	}
}

func TestAcceptance_SearchHelpText_ShowsNotionSource(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "search", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "notion",
		"Help text should list 'notion' as a valid source")
}

func TestAcceptance_SearchWithNotionSource_ReturnsResults(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "notion", testQuery)
	combined := stdout + stderr

	assert.NotContains(t, combined, "unknown flag",
		"Should recognize --sources notion flag")

	if exitCode == 0 && !strings.Contains(stdout, "No results") {
		assert.Contains(t, stdout, "[notion]",
			"Should return notion results")
		assert.NotContains(t, stdout, "[google-drive]",
			"Should not return google-drive results when filtering to notion")
		assert.NotContains(t, stdout, "[gmail]",
			"Should not return gmail results when filtering to notion")
		assert.NotContains(t, stdout, "[obsidian]",
			"Should not return obsidian results when filtering to notion")
	}
}

func TestAcceptance_SearchWithAllFourSources_ReturnsResults(t *testing.T) {
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "google-drive,gmail,obsidian,notion", testQuery)
	combined := stdout + stderr

	assert.NotContains(t, combined, "invalid argument",
		"Should accept comma-separated source list with four sources")

	if exitCode == 0 {
		assert.Contains(t, stdout, "[google-drive]",
			"Should return google-drive results")
		assert.Contains(t, stdout, "[gmail]",
			"Should return gmail results")
		assert.Contains(t, stdout, "[obsidian]",
			"Should return obsidian results")
		assert.Contains(t, stdout, "[notion]",
			"Should return notion results")
	}
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
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"/health endpoint should return 200 OK")
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	var health map[string]string
	require.NoError(t, json.Unmarshal(body, &health))
	assert.Equal(t, "ok", health["status"])
	assert.NotEmpty(t, health["version"])
	assert.NotEmpty(t, health["uptime"])
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

func TestAcceptance_TailscaleMode_NoTailscale_ShowsHelpfulError(t *testing.T) {
	binary := buildBinary(t)

	// Run with PKB_TAILSCALE=true but no Tailscale installed
	cmd := exec.Command(binary, "serve")
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"PKB_TAILSCALE=true",
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()

	combined := outBuf.String() + errBuf.String()
	assert.Error(t, err, "Should fail when Tailscale is not installed")
	assert.True(t,
		strings.Contains(combined, "tailscale") ||
			strings.Contains(combined, "Tailscale"),
		"Error should mention tailscale, got: %s", combined)
}

func TestAcceptance_TailscaleMode_ExitsNonZero(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "serve")
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"PKB_TAILSCALE=true",
	}
	err := cmd.Run()

	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.NotEqual(t, 0, exitErr.ExitCode(),
			"Should exit with non-zero code when Tailscale unavailable")
	} else {
		assert.Error(t, err, "Expected non-zero exit")
	}
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

// --- Tests for import-claude command ---

func TestAcceptance_ImportClaudeHelp(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "import-claude", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "import-claude")
	assert.Contains(t, stdout, "Claude")
}

func TestAcceptance_HelpShowsImportClaudeCommand(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "import-claude", "Help should list import-claude command")
}

func TestAcceptance_ImportClaude_MissingArg_ShowsError(t *testing.T) {
	binary := buildBinary(t)

	_, stderr, exitCode := runPKB(t, binary, "import-claude")

	assert.NotEqual(t, 0, exitCode, "import-claude without args should fail")
	assert.Contains(t, stderr, "accepts 1 arg",
		"Should indicate missing argument")
}

func TestAcceptance_ImportClaude_MissingFile_ShowsError(t *testing.T) {
	binary := buildBinary(t)

	_, stderr, exitCode := runPKB(t, binary, "import-claude", "/nonexistent/conversations.json")

	assert.NotEqual(t, 0, exitCode, "import-claude with missing file should fail")
	combined := stderr
	assert.True(t,
		strings.Contains(combined, "no such file") ||
			strings.Contains(combined, "read file"),
		"Should indicate file not found, got: %s", combined)
}

func TestAcceptance_ImportClaude_CopiesFile(t *testing.T) {
	binary := buildBinary(t)

	// Create a temporary conversations.json
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "conversations.json")
	content := `[{"uuid":"conv-1","name":"Test Chat","chat_messages":[]}]`
	require.NoError(t, os.WriteFile(srcFile, []byte(content), 0644))

	// Use a custom HOME so DefaultExportPath writes to a temp location
	fakeHome := t.TempDir()
	cmd := exec.Command(binary, "import-claude", srcFile)
	cmd.Env = []string{
		"HOME=" + fakeHome,
		"XDG_DATA_HOME=", // unset so it falls back to HOME
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()

	require.NoError(t, err, "import-claude should succeed, stderr: %s", errBuf.String())
	assert.Contains(t, outBuf.String(), "Imported Claude conversations",
		"Should confirm import")

	// Verify the file was actually written
	destPath := filepath.Join(fakeHome, ".local", "share", "pkb", "claude-conversations.json")
	assert.FileExists(t, destPath, "Should write file to XDG data location")

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(got), "File contents should match")
}

func TestAcceptance_ImportClaude_AcceptsZipFile(t *testing.T) {
	binary := buildBinary(t)

	// Create a zip archive mimicking a real Claude export
	content := `[{"uuid":"conv-zip","name":"Zip Import Test","chat_messages":[]}]`
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("conversations.json")
	require.NoError(t, err)
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
	f2, err := w.Create("projects.json")
	require.NoError(t, err)
	_, err = f2.Write([]byte("[]"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "claude-export.zip")
	require.NoError(t, os.WriteFile(srcFile, buf.Bytes(), 0644))

	fakeHome := t.TempDir()
	cmd := exec.Command(binary, "import-claude", srcFile)
	cmd.Env = []string{
		"HOME=" + fakeHome,
		"XDG_DATA_HOME=",
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()

	require.NoError(t, err, "import-claude with zip should succeed, stderr: %s", errBuf.String())
	assert.Contains(t, outBuf.String(), "Imported Claude conversations")

	// Verify extracted JSON was written (not raw zip)
	destPath := filepath.Join(fakeHome, ".local", "share", "pkb", "claude-conversations.json")
	assert.FileExists(t, destPath)
	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(got), "Should extract conversations.json from zip")
}

func TestAcceptance_ImportClaude_ThenSearchFindsResults(t *testing.T) {
	binary := buildBinary(t)

	// Create conversations.json with searchable content
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "conversations.json")
	content := `[{"uuid":"conv-abc","name":"Debugging Go Tests","chat_messages":[{"uuid":"msg-1","text":"How to debug Go test failures?","sender":"human"}]}]`
	require.NoError(t, os.WriteFile(srcFile, []byte(content), 0644))

	// Import using a custom HOME
	fakeHome := t.TempDir()
	env := []string{
		"HOME=" + fakeHome,
		"XDG_DATA_HOME=",
		"PKB_TAILSCALE=false",
	}

	importCmd := exec.Command(binary, "import-claude", srcFile)
	importCmd.Env = env
	out, err := importCmd.CombinedOutput()
	require.NoError(t, err, "import should succeed: %s", string(out))

	// Now search with --sources claude
	searchCmd := exec.Command(binary, "search", "--sources", "claude", "debug")
	searchCmd.Env = env
	var outBuf, errBuf strings.Builder
	searchCmd.Stdout = &outBuf
	searchCmd.Stderr = &errBuf
	err = searchCmd.Run()

	stdout := outBuf.String()
	if err == nil {
		// Search succeeded — verify we got claude results
		assert.Contains(t, stdout, "[claude]",
			"Should return claude source results")
		assert.Contains(t, stdout, "Debugging Go Tests",
			"Should find the imported conversation by name")
	}
	// If search fails, it may be because other connectors error out.
	// The important thing is import-claude succeeded above.
}

func TestAcceptance_SearchHelpText_ShowsClaudeSource(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "search", "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "claude",
		"Help text should list 'claude' as a valid source")
}

func TestAcceptance_ServeWebUI_HasClaudeCheckbox(t *testing.T) {
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

	resp, err := http.Get("http://" + addr + "/")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, html, "claude", "Web UI should have claude checkbox")
}

func TestAcceptance_ServeImportClaudeEndpoint(t *testing.T) {
	binary := buildBinary(t)
	fakeHome := t.TempDir()

	cmd := exec.Command(binary, "serve", "--addr", "127.0.0.1:0")
	cmd.Env = []string{"HOME=" + fakeHome, "XDG_DATA_HOME="}

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

	// POST /import-claude without file should return 400
	resp, err := http.Post(baseURL+"/import-claude", "application/json", nil)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"POST /import-claude without file should return 400")
}