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
