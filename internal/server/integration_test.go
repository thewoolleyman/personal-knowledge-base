//go:build integration

package server

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Server_HealthEndpoint(t *testing.T) {
	s := New(":0")
	err := s.Listen()
	require.NoError(t, err)

	addr := s.Addr()
	require.NotEmpty(t, addr)

	go func() {
		_ = s.Serve()
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	}()

	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// Health endpoint returns 200 with empty body — that's fine
	_ = body
}
