//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcceptance_HealthEndpoint_ReturnsJSON builds the real binary, starts
// the server, and verifies that GET /health returns a well-formed JSON
// response including status, version, and uptime fields.
func TestAcceptance_HealthEndpoint_ReturnsJSON(t *testing.T) {
	binary := buildBinary(t)

	// Start the server on an ephemeral port.
	cmd := exec.Command(binary, "serve", "--addr", ":0")
	cmd.Env = append(cmd.Environ(), "PKB_TAILSCALE=false")

	// Capture stdout to find the listen address.
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// Read the "Listening on ..." line to get the address.
	buf := make([]byte, 512)
	n, err := stdout.Read(buf)
	require.NoError(t, err)
	output := string(buf[:n])

	// Extract address from "Listening on 127.0.0.1:XXXXX\n"
	var addr string
	_, err = fmt.Sscanf(output, "Listening on %s", &addr)
	require.NoError(t, err, "could not parse listen address from: %q", output)

	// Give the server a moment to be ready for requests.
	time.Sleep(100 * time.Millisecond)

	// Hit the /health endpoint.
	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var health map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&health))

	// Core fields must be present.
	assert.Contains(t, health, "status")
	assert.Contains(t, health, "version")
	assert.Contains(t, health, "uptime")

	// Status should be "ok" or "degraded" (degraded if no token is configured).
	status := health["status"].(string)
	assert.True(t, status == "ok" || status == "degraded",
		"status should be 'ok' or 'degraded', got %q", status)
}
