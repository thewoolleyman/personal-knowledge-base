//go:build acceptance

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

func repoRoot(t *testing.T) string {
	t.Helper()
	// Walk up from test dir to find VERSION file
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no VERSION file found)")
		}
		dir = parent
	}
}

func TestBuildReleaseScript(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, "scripts", "build-release.sh")

	// Use a test version to avoid conflating with real releases
	testVersion := "0.0.0-test"

	cmd := exec.Command(script, testVersion)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build-release.sh failed:\n%s", out)

	// Verify all expected zip files exist
	expectedPlatforms := []string{
		"linux-amd64", "linux-arm64",
		"darwin-amd64", "darwin-arm64",
		"windows-amd64",
	}
	distDir := filepath.Join(root, "dist")
	for _, plat := range expectedPlatforms {
		zipFile := filepath.Join(distDir, "pkb-"+plat+".zip")
		_, err := os.Stat(zipFile)
		assert.NoError(t, err, "expected release artifact %s to exist", zipFile)
	}

	// Clean up
	t.Cleanup(func() {
		os.RemoveAll(distDir)
	})
}

func TestValidateReleaseScript(t *testing.T) {
	root := repoRoot(t)
	buildScript := filepath.Join(root, "scripts", "build-release.sh")
	validateScript := filepath.Join(root, "scripts", "validate-release.sh")

	testVersion := "0.0.0-test"

	// First build
	cmd := exec.Command(buildScript, testVersion)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build-release.sh failed:\n%s", out)

	// Then validate
	cmd = exec.Command(validateScript, testVersion)
	cmd.Dir = root
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "validate-release.sh failed:\n%s", out)

	assert.Contains(t, string(out), "Validation passed")
	assert.Contains(t, string(out), testVersion)

	// Clean up
	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(root, "dist"))
	})
}

func TestValidateReleaseScriptFailsWithoutArtifacts(t *testing.T) {
	root := repoRoot(t)
	validateScript := filepath.Join(root, "scripts", "validate-release.sh")

	// Ensure dist is clean
	distDir := filepath.Join(root, "dist")
	os.RemoveAll(distDir)

	cmd := exec.Command(validateScript, "0.0.0-nonexistent")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	assert.Error(t, err, "validate-release.sh should fail when no artifacts exist")
	assert.True(t, strings.Contains(string(out), "FAIL"), "expected FAIL in output:\n%s", out)
}

func TestMakeReleaseTargets(t *testing.T) {
	root := repoRoot(t)

	// Test make release-build
	cmd := exec.Command("make", "release-build")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "make release-build failed:\n%s", out)

	// Test make release-validate
	cmd = exec.Command("make", "release-validate")
	cmd.Dir = root
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "make release-validate failed:\n%s", out)

	assert.Contains(t, string(out), "Validation passed")

	// Clean up
	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(root, "dist"))
	})
}
