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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// --- Tests mirror what the README tells a human to do ---

func TestAcceptance_HelpShowsSearchCommand(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runPKB(t, binary, "--help")

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "search")
	assert.Contains(t, stdout, "Personal Knowledge Base")
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
