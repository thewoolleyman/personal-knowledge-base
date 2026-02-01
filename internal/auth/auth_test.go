package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestFlow_Run_Success(t *testing.T) {
	// Set up a mock token exchange server.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"Bearer"}`)
	}))
	defer tokenServer.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: tokenServer.URL,
		},
	}

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			// Simulate the browser redirect: parse the auth URL, extract
			// the redirect_uri, then hit it with a code.
			go func() {
				parsed, err := neturl.Parse(rawURL)
				if err != nil {
					return
				}
				redirectURI := parsed.Query().Get("redirect_uri")
				//nolint:gosec // test-only HTTP request
				resp, err := http.Get(redirectURI + "?code=test-code")
				if err == nil {
					resp.Body.Close()
				}
			}()
			return nil
		},
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
}

func TestFlow_Run_NoCodeInCallback(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			go func() {
				parsed, err := neturl.Parse(rawURL)
				if err != nil {
					return
				}
				redirectURI := parsed.Query().Get("redirect_uri")
				// Hit callback WITHOUT a code parameter.
				//nolint:gosec // test-only HTTP request
				resp, err := http.Get(redirectURI)
				if err == nil {
					resp.Body.Close()
				}
			}()
			return nil
		},
	}

	_, err := flow.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no code in callback")
}

func TestFlow_Run_ContextCancelled(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			// Cancel immediately -- do not simulate a callback.
			cancel()
			return nil
		},
	}

	_, err := flow.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFlow_Run_BrowserOpenError(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			return fmt.Errorf("browser not found")
		},
	}

	_, err := flow.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open browser")
	assert.Contains(t, err.Error(), "browser not found")
}

func TestFlow_Run_ExchangeError(t *testing.T) {
	// Token server returns an error response.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant"}`)
	}))
	defer tokenServer.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: tokenServer.URL,
		},
	}

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			go func() {
				parsed, err := neturl.Parse(rawURL)
				if err != nil {
					return
				}
				redirectURI := parsed.Query().Get("redirect_uri")
				//nolint:gosec // test-only HTTP request
				resp, err := http.Get(redirectURI + "?code=bad-code")
				if err == nil {
					resp.Body.Close()
				}
			}()
			return nil
		},
	}

	_, err := flow.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exchange code")
}

func TestFlow_Run_CallbackRendersSuccess(t *testing.T) {
	// Verify the callback response body says "Authorization successful".
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"tok","token_type":"Bearer"}`)
	}))
	defer tokenServer.Close()

	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: tokenServer.URL,
		},
	}

	bodyCh := make(chan string, 1)
	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			go func() {
				parsed, _ := neturl.Parse(rawURL)
				redirectURI := parsed.Query().Get("redirect_uri")
				//nolint:gosec // test-only HTTP request
				resp, err := http.Get(redirectURI + "?code=test-code")
				if err == nil {
					defer resp.Body.Close()
					buf := make([]byte, 1024)
					n, _ := resp.Body.Read(buf)
					bodyCh <- string(buf[:n])
				}
			}()
			return nil
		},
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "tok", token.AccessToken)

	select {
	case body := <-bodyCh:
		assert.Contains(t, body, "Authorization successful")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for callback body")
	}
}

func TestFlow_Run_ListenerError(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	flow := &Flow{
		Config:     cfg,
		OpenURL:    func(rawURL string) error { return nil },
		ListenAddr: "999.999.999.999:0", // invalid address forces listen error
	}

	_, err := flow.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start callback server")
}
