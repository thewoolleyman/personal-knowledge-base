package tailscale

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveIP_Success(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "100.64.0.1\n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	ip, err := ResolveIP()
	require.NoError(t, err)
	assert.Equal(t, "100.64.0.1", ip)
}

func TestResolveIP_TrimsWhitespace(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "  100.100.1.2  \n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	ip, err := ResolveIP()
	require.NoError(t, err)
	assert.Equal(t, "100.100.1.2", ip)
}

func TestResolveIP_TailscaleNotRunning(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "", fmt.Errorf("exit status 1")
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	_, err := ResolveIP()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tailscale")
}

func TestResolveIP_EmptyOutput(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	_, err := ResolveIP()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestResolveAddr_CombinesIPAndPort(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "100.64.0.1\n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	addr, err := ResolveAddr(":8080")
	require.NoError(t, err)
	assert.Equal(t, "100.64.0.1:8080", addr)
}

func TestResolveAddr_PreservesCustomPort(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "100.64.0.1\n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	addr, err := ResolveAddr(":9090")
	require.NoError(t, err)
	assert.Equal(t, "100.64.0.1:9090", addr)
}

func TestResolveAddr_ExtractsPortFromFullAddr(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "100.64.0.1\n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	addr, err := ResolveAddr("0.0.0.0:8080")
	require.NoError(t, err)
	assert.Equal(t, "100.64.0.1:8080", addr)
}

func TestResolveAddr_InvalidAddr_DefaultsTo8080(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "100.64.0.1\n", nil
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	addr, err := ResolveAddr("no-colon-here")
	require.NoError(t, err)
	assert.Equal(t, "100.64.0.1:8080", addr)
}

func TestResolveAddr_ErrorPropagates(t *testing.T) {
	orig := runTailscaleIP
	runTailscaleIP = func() (string, error) {
		return "", fmt.Errorf("not running")
	}
	t.Cleanup(func() { runTailscaleIP = orig })

	_, err := ResolveAddr(":8080")
	require.Error(t, err)
}
