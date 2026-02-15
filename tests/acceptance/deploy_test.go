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

// TestDeployServiceFileExists verifies the systemd service template is shipped.
func TestDeployServiceFileExists(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "deploy", "pkb.service")
	require.FileExists(t, path, "deploy/pkb.service must exist")

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "ExecStart=%h/.local/bin/pkb serve")
	assert.Contains(t, content, "EnvironmentFile=%h/.config/pkb/.env")
	assert.Contains(t, content, "Restart=on-failure")
}

// TestMakeDeployTargetsExist verifies the deploy-related make targets are defined.
func TestMakeDeployTargetsExist(t *testing.T) {
	root := projectRoot(t)

	// `make -n <target>` (dry-run) exits 0 if the target exists, non-zero otherwise.
	targets := []string{"deploy", "deploy-local", "deploy-status", "deploy-logs", "deploy-setup"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			cmd := exec.Command("make", "-n", target)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			assert.NoError(t, err, "make target %q should exist; output: %s", target, string(out))
		})
	}
}

// TestCICDHasDeployJob verifies the CI/CD workflow includes a deploy job.
func TestCICDHasDeployJob(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, ".github", "workflows", "ci-cd.yml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.True(t, strings.Contains(content, "deploy:"), "ci-cd.yml must contain a deploy job")
	assert.True(t, strings.Contains(content, "Deploy to Production"), "ci-cd.yml must have Deploy to Production job name")
}
