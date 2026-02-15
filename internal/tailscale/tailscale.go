package tailscale

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// runTailscaleIP executes "tailscale ip -4" and returns stdout. Overridden in tests.
var runTailscaleIP = func() (string, error) {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	return string(out), err
}

// ResolveIP returns the machine's Tailscale IPv4 address.
func ResolveIP() (string, error) {
	out, err := runTailscaleIP()
	if err != nil {
		return "", fmt.Errorf("failed to get tailscale IP (is tailscale running?): %w", err)
	}
	ip := strings.TrimSpace(out)
	if ip == "" {
		return "", fmt.Errorf("tailscale returned empty IP address")
	}
	return ip, nil
}

// ResolveAddr takes a listen address like ":8080" or "0.0.0.0:8080" and
// returns "<tailscale-ip>:<port>".
func ResolveAddr(listenAddr string) (string, error) {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		port = "8080"
	}
	ip, err := ResolveIP()
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(ip, port), nil
}
