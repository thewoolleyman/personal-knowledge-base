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

func TestNew_CreatesServer(t *testing.T) {
	s := New(":0")
	assert.NotNil(t, s)
}

func TestServer_Addr_EmptyBeforeListen(t *testing.T) {
	s := New(":0")
	assert.Empty(t, s.Addr())
}

func TestServer_Serve_ErrorWithoutListen(t *testing.T) {
	s := New(":0")
	err := s.Serve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must call Listen before Serve")
}

func TestServer_Listen_ErrorPortInUse(t *testing.T) {
	s1 := New(":0")
	require.NoError(t, s1.Listen())
	t.Cleanup(func() { s1.listener.Close() })

	s2 := New(s1.Addr())
	err := s2.Listen()
	assert.Error(t, err)
}

func TestServer_StartsAndStops(t *testing.T) {
	s := New(":0")

	err := s.Listen()
	require.NoError(t, err)

	addr := s.Addr()
	require.NotEmpty(t, addr)

	go func() {
		_ = s.Serve()
	}()

	// Verify it responds
	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServer_Handle_RegistersRoute(t *testing.T) {
	s := New(":0")

	// Register a custom handler via the Handle method.
	s.Handle("GET /custom", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("custom-ok"))
	}))

	require.NoError(t, s.Listen())
	go func() { _ = s.Serve() }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	// Verify the custom handler responds.
	resp, err := http.Get("http://" + s.Addr() + "/custom")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "custom-ok", string(body))

	// Verify /health still works too.
	resp2, err := http.Get("http://" + s.Addr() + "/health")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}
