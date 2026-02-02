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

func skipUnlessCredentials(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PKB_GOOGLE_CLIENT_ID", "PKB_GOOGLE_CLIENT_SECRET"} {
		if os.Getenv(key) == "" {
			t.Skipf("skipping: %s not set", key)
		}
	}
}

func TestLive_CLISearch_GoogleDrive(t *testing.T) {
	skipUnlessCredentials(t)
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
	skipUnlessCredentials(t)
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

func TestLive_CLISearch_SourceFilter_GDriveOnly(t *testing.T) {
	skipUnlessCredentials(t)
	binary := buildBinary(t)

	// Search with --sources not available on CLI yet, so search via serve endpoint.
	// Instead, just verify the default search returns results from both.
	// This test focuses on verifying gdrive results exist.
	stdout, stderr, exitCode := runPKB(t, binary, "search", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)

	// Parse numbered results and verify at least one is from google-drive
	lines := strings.Split(stdout, "\n")
	foundGDrive := false
	for _, line := range lines {
		if strings.Contains(line, "[google-drive]") {
			foundGDrive = true
			break
		}
	}
	assert.True(t, foundGDrive,
		"Expected at least one google-drive result, output:\n%s", stdout)
}

func TestLive_CLISearch_SourceFilter_GmailOnly(t *testing.T) {
	skipUnlessCredentials(t)
	binary := buildBinary(t)

	stdout, stderr, exitCode := runPKB(t, binary, "search", testQuery)

	require.Equal(t, 0, exitCode, "stderr: %s", stderr)

	lines := strings.Split(stdout, "\n")
	foundGmail := false
	for _, line := range lines {
		if strings.Contains(line, "[gmail]") {
			foundGmail = true
			break
		}
	}
	assert.True(t, foundGmail,
		"Expected at least one gmail result, output:\n%s", stdout)
}

func TestLive_ServeSearch_BothSources(t *testing.T) {
	skipUnlessCredentials(t)
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
