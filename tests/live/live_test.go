//go:build live

// Package live contains top-of-the-pyramid tests that hit real external APIs.
// These tests build the actual binary and run it as a subprocess, verifying
// that end-to-end search works against real Google Drive and Gmail.
//
// Prerequisites:
//   - PKB_GOOGLE_CLIENT_ID and PKB_GOOGLE_CLIENT_SECRET set
//   - PKB_TOKEN_PATH pointing to a valid OAuth token
//   - A file named "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE" in Google Drive
//   - An email with "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE" in Gmail
//
// Run: go test -tags=live -v -timeout=60s ./tests/live/
package live

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testQuery = "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE"

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../..")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "go.mod"), "Could not find project root")
	return dir
}

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

func requireCredentials(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PKB_GOOGLE_CLIENT_ID", "PKB_GOOGLE_CLIENT_SECRET"} {
		if os.Getenv(key) == "" {
			t.Fatalf("%s not set — add it to your .env file", key)
		}
	}
}

func TestLive_CLISearch_GoogleDrive(t *testing.T) {
	requireCredentials(t)
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", testQuery)

	require.Equal(t, 0, exitCode,
		"search should succeed, stderr: %s", stderr)
	require.NotEmpty(t, stdout, "Expected results in stdout")

	assert.Contains(t, stdout, "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE",
		"Result should contain the test page title")
	assert.Contains(t, stdout, "[google-drive]",
		"Result should show google-drive source")
}

func TestLive_CLISearch_Gmail(t *testing.T) {
	requireCredentials(t)
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", testQuery)

	require.Equal(t, 0, exitCode,
		"search should succeed, stderr: %s", stderr)
	require.NotEmpty(t, stdout, "Expected results in stdout")

	assert.Contains(t, stdout, "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE",
		"Result should contain the test email subject")
	assert.Contains(t, stdout, "[gmail]",
		"Result should show gmail source")
}

func TestLive_CLISearch_SourceFilter_GoogleDriveOnly(t *testing.T) {
	requireCredentials(t)
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "google-drive", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)
	assert.Contains(t, stdout, "[google-drive]",
		"Expected google-drive results when filtering by --sources google-drive")
	assert.NotContains(t, stdout, "[gmail]",
		"Should not contain gmail results when filtering to google-drive only")
}

func TestLive_CLISearch_SourceFilter_GmailOnly(t *testing.T) {
	requireCredentials(t)
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "gmail", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)
	assert.Contains(t, stdout, "[gmail]",
		"Expected gmail results when filtering by --sources gmail")
	assert.NotContains(t, stdout, "[google-drive]",
		"Should not contain google-drive results when filtering to gmail only")
}

func TestLive_ServeSearch_BothSources(t *testing.T) {
	requireCredentials(t)
	binary := buildBinary(t)

	// Use the serve command's search endpoint via the embedded server in CLI.
	// The CLI search command already goes through the HTTP API, so this
	// effectively tests the full stack.
	stdout, stderr, exitCode := runPKB(t, binary, "search", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)

	// Verify we get results from both sources
	assert.Contains(t, stdout, "[google-drive]", "Expected google-drive results")
	assert.Contains(t, stdout, "[gmail]", "Expected gmail results")

	// Verify the output has the expected format: numbered results
	assert.Contains(t, stdout, "1.", "Expected at least one numbered result")

	// Verify results contain URLs
	assert.Contains(t, stdout, "http", "Expected URLs in results")

	fmt.Fprintf(os.Stderr, "\n--- Live search output ---\n%s\n", stdout)
}

func TestLive_CLISearch_NotionSource(t *testing.T) {
	requireCredentials(t)
	if os.Getenv("PKB_NOTION_TOKEN") == "" {
		t.Fatalf("PKB_NOTION_TOKEN not set — add it to your .env file")
	}
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "notion", testQuery)

	require.Equal(t, 0, exitCode,
		"search should succeed, stderr: %s", stderr)
	require.NotEmpty(t, stdout, "Expected results in stdout")

	assert.Contains(t, stdout, "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE (Notion)",
		"Notion source should find the test page")
	assert.Contains(t, stdout, "[notion]",
		"Result should show notion source tag")

	// Verify only one result is returned (no fuzzy false positives)
	lines := strings.Split(strings.TrimSpace(stdout), "\n\n")
	assert.Len(t, lines, 1,
		"Expected exactly one result block; Notion search should filter non-matching pages")
}

func TestLive_CLISearch_NotionSource_OnlyExactMatch(t *testing.T) {
	requireCredentials(t)
	if os.Getenv("PKB_NOTION_TOKEN") == "" {
		t.Fatalf("PKB_NOTION_TOKEN not set — add it to your .env file")
	}
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "notion", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)

	// Count numbered results (lines starting with "N.")
	resultCount := 0
	for _, line := range strings.Split(stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] >= '1' && trimmed[0] <= '9' && strings.Contains(trimmed, ".") {
			resultCount++
		}
	}
	assert.Equal(t, 1, resultCount,
		"Should return exactly 1 Notion result, got stdout:\n%s", stdout)
}

func TestLive_CLISearch_ObsidianSource(t *testing.T) {
	requireCredentials(t)
	if os.Getenv("PKB_OBSIDIAN_FOLDER_ID") == "" {
		t.Fatalf("PKB_OBSIDIAN_FOLDER_ID not set — add it to your .env file")
	}
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", "--sources", "obsidian", testQuery)

	require.Equal(t, 0, exitCode,
		"search should succeed, stderr: %s", stderr)
	require.NotEmpty(t, stdout, "Expected results in stdout")

	assert.Contains(t, stdout, "PERSONAL_KNOWLEDGE_BASE_TEST_PAGE_DO_NOT_DELETE (Obsidian)",
		"Obsidian source should find the test page in the Obsidian vault subfolder")
	assert.Contains(t, stdout, "[obsidian]",
		"Result should show obsidian source tag")
}
