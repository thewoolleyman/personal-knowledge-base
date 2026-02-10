//go:build acceptance

package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// Verify all expected zip files exist (versioned filenames)
	expectedPlatforms := []string{
		"linux-amd64", "linux-arm64",
		"darwin-amd64", "darwin-arm64",
		"windows-amd64",
	}
	distDir := filepath.Join(root, "dist")
	for _, plat := range expectedPlatforms {
		zipFile := filepath.Join(distDir, "pkb-"+plat+"-v"+testVersion+".zip")
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

func TestReleaseZipContainsSingleBinary(t *testing.T) {
	root := repoRoot(t)
	buildScript := filepath.Join(root, "scripts", "build-release.sh")

	testVersion := "0.0.0-ziptest"

	// Build only for current platform to keep test fast
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	cmd := exec.Command(buildScript, testVersion)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TARGETS="+goos+"/"+goarch)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build-release.sh failed:\n%s", out)

	distDir := filepath.Join(root, "dist")
	t.Cleanup(func() { os.RemoveAll(distDir) })

	// Extract the zip and verify it contains a subdirectory with the binary
	archiveName := "pkb-" + goos + "-" + goarch + "-v" + testVersion
	zipFile := filepath.Join(distDir, archiveName+".zip")
	require.FileExists(t, zipFile)

	tmpDir := t.TempDir()
	unzipCmd := exec.Command("unzip", "-o", zipFile, "-d", tmpDir)
	out, err = unzipCmd.CombinedOutput()
	require.NoError(t, err, "unzip failed:\n%s", out)

	// Verify subdirectory exists
	subDir := filepath.Join(tmpDir, archiveName)
	assert.DirExists(t, subDir, "zip should extract to subdirectory %s", archiveName)

	expectedBinary := "pkb"
	if goos == "windows" {
		expectedBinary = "pkb.exe"
	}

	binPath := filepath.Join(subDir, expectedBinary)
	assert.FileExists(t, binPath, "subdirectory should contain binary named %s", expectedBinary)

	// Verify it's executable (not applicable on Windows)
	if goos != "windows" {
		info, err := os.Stat(binPath)
		require.NoError(t, err)
		assert.NotZero(t, info.Mode()&0111, "Binary should be executable")
	}

	// Verify it runs and reports the correct version
	runCmd := exec.Command(binPath, "version")
	out, err = runCmd.CombinedOutput()
	require.NoError(t, err, "binary should run: %s", out)
	assert.Contains(t, string(out), testVersion)
}

func TestReleaseZipMacOSCodesigned(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification only runs on macOS")
	}

	root := repoRoot(t)
	buildScript := filepath.Join(root, "scripts", "build-release.sh")

	testVersion := "0.0.0-signtest"

	cmd := exec.Command(buildScript, testVersion)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TARGETS=darwin/"+runtime.GOARCH)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build-release.sh failed:\n%s", out)

	distDir := filepath.Join(root, "dist")
	t.Cleanup(func() { os.RemoveAll(distDir) })

	archiveName := "pkb-darwin-" + runtime.GOARCH + "-v" + testVersion
	zipFile := filepath.Join(distDir, archiveName+".zip")
	require.FileExists(t, zipFile)

	tmpDir := t.TempDir()
	unzipCmd := exec.Command("unzip", "-o", zipFile, "-d", tmpDir)
	out, err = unzipCmd.CombinedOutput()
	require.NoError(t, err, "unzip failed:\n%s", out)

	binPath := filepath.Join(tmpDir, archiveName, "pkb")

	// Verify ad-hoc code signature
	verifyCmd := exec.Command("codesign", "--verify", "--verbose", binPath)
	out, err = verifyCmd.CombinedOutput()
	require.NoError(t, err, "codesign verification failed:\n%s", out)
	assert.Contains(t, string(out), "valid on disk",
		"Binary should have valid code signature")

	// Verify it's ad-hoc signed (not identity-signed)
	displayCmd := exec.Command("codesign", "-d", "--verbose", binPath)
	out, err = displayCmd.CombinedOutput()
	require.NoError(t, err, "codesign display failed:\n%s", out)
	assert.Contains(t, string(out), "adhoc",
		"Binary should be ad-hoc signed")
}

func TestReleaseInstallFlow(t *testing.T) {
	// This test mirrors the installation flow documented in the README:
	// 1. Download zip (we build it here instead)
	// 2. Extract binary
	// 3. Remove macOS quarantine (if on macOS)
	// 4. Make executable
	// 5. Run "pkb version"

	root := repoRoot(t)
	buildScript := filepath.Join(root, "scripts", "build-release.sh")

	testVersion := "0.0.0-installtest"

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	cmd := exec.Command(buildScript, testVersion)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TARGETS="+goos+"/"+goarch)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build-release.sh failed:\n%s", out)

	distDir := filepath.Join(root, "dist")
	t.Cleanup(func() { os.RemoveAll(distDir) })

	archiveName := "pkb-" + goos + "-" + goarch + "-v" + testVersion
	zipFile := filepath.Join(distDir, archiveName+".zip")
	installDir := t.TempDir()

	// Step 2: Extract (zip contains a subdirectory)
	unzipCmd := exec.Command("unzip", "-o", zipFile, "-d", installDir)
	out, err = unzipCmd.CombinedOutput()
	require.NoError(t, err, "unzip failed:\n%s", out)

	expectedBinary := "pkb"
	if goos == "windows" {
		expectedBinary = "pkb.exe"
	}
	binPath := filepath.Join(installDir, archiveName, expectedBinary)

	// Step 3: On macOS, simulate quarantine and remove it
	if goos == "darwin" {
		// Add quarantine attribute (simulates browser download)
		xattrCmd := exec.Command("xattr", "-w", "com.apple.quarantine",
			"0082;00000000;Safari;", binPath)
		out, err = xattrCmd.CombinedOutput()
		require.NoError(t, err, "failed to set quarantine: %s", out)

		// Remove quarantine (what the README tells users to do)
		removeCmd := exec.Command("xattr", "-d", "com.apple.quarantine", binPath)
		out, err = removeCmd.CombinedOutput()
		require.NoError(t, err, "failed to remove quarantine: %s", out)

		// Verify quarantine is gone
		listCmd := exec.Command("xattr", binPath)
		out, _ = listCmd.CombinedOutput()
		assert.NotContains(t, string(out), "com.apple.quarantine",
			"quarantine should be removed")
	}

	// Step 4: Make executable
	require.NoError(t, os.Chmod(binPath, 0755))

	// Step 5: Run and verify
	runCmd := exec.Command(binPath, "version")
	out, err = runCmd.CombinedOutput()
	require.NoError(t, err, "binary should run after install flow: %s", out)
	assert.Contains(t, string(out), testVersion,
		"binary should report the correct version")
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
