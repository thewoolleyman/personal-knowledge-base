package auth

import (
	"bytes"
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

func TestFlow_Run_AuthURL_ForcesConsentAndOffline(t *testing.T) {
	// Verify the auth URL includes prompt=consent and access_type=offline
	// so Google always returns a refresh token.
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var capturedURL string
	ctx, cancel := context.WithCancel(context.Background())

	flow := &Flow{
		Config: cfg,
		OpenURL: func(rawURL string) error {
			capturedURL = rawURL
			cancel() // cancel so flow doesn't block
			return nil
		},
	}

	_, _ = flow.Run(ctx) // will return ctx.Canceled

	parsed, err := neturl.Parse(capturedURL)
	require.NoError(t, err)

	assert.Equal(t, "offline", parsed.Query().Get("access_type"),
		"auth URL must include access_type=offline for refresh token")
	assert.Equal(t, "consent", parsed.Query().Get("prompt"),
		"auth URL must include prompt=consent to always return refresh token")
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

func TestFlow_Run_BrowserOpenFallback(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	flow := &Flow{
		Config: cfg,
		Out:    &buf,
		OpenURL: func(rawURL string) error {
			cancel() // cancel so flow doesn't block waiting for callback
			return fmt.Errorf("browser not found")
		},
	}

	_, err := flow.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, buf.String(), "Could not open browser automatically")
	assert.Contains(t, buf.String(), "http://example.com/auth")
	assert.Contains(t, buf.String(), "prompt=consent")
}

func TestFlow_Run_BrowserOpenFallback_NilOut(t *testing.T) {
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
		// Out is intentionally nil to exercise the os.Stderr fallback.
		OpenURL: func(rawURL string) error {
			cancel()
			return fmt.Errorf("no display")
		},
	}

	_, err := flow.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
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

func TestFlow_Run_ManualCodeEntry(t *testing.T) {
	// When the callback server is not reachable (remote machine), the user
	// can paste the auth code from the browser URL bar via stdin.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"manual-token","token_type":"Bearer"}`)
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

	var outBuf bytes.Buffer
	// Simulate user typing the code into stdin.
	inBuf := bytes.NewBufferString("manual-auth-code\n")

	flow := &Flow{
		Config: cfg,
		Out:    &outBuf,
		In:     inBuf,
		OpenURL: func(rawURL string) error {
			// Browser open fails (remote machine) -- do NOT simulate callback.
			return fmt.Errorf("no display")
		},
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "manual-token", token.AccessToken)
	assert.Contains(t, outBuf.String(), "Paste URL or code")
}

func TestFlow_Run_ManualCodeEntry_TrimsWhitespace(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the code arrives trimmed by checking the form value.
		if r.FormValue("code") == "trimmed-code" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"trimmed-token","token_type":"Bearer"}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":"bad code: %s"}`, r.FormValue("code"))
		}
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

	var outBuf bytes.Buffer
	inBuf := bytes.NewBufferString("  trimmed-code  \n")

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      inBuf,
		OpenURL: func(_ string) error { return fmt.Errorf("no browser") },
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "trimmed-token", token.AccessToken)
}

func TestFlow_Run_ManualCodeEntry_ExtractsCodeFromURL(t *testing.T) {
	// User might paste the entire callback URL instead of just the code.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"url-token","token_type":"Bearer"}`)
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

	var outBuf bytes.Buffer
	// Simulate user pasting the full callback URL.
	inBuf := bytes.NewBufferString("http://127.0.0.1:12345/callback?state=state&code=extracted-code&scope=foo\n")

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      inBuf,
		OpenURL: func(_ string) error { return fmt.Errorf("no browser") },
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "url-token", token.AccessToken)
}

func TestFlow_Run_ManualCodeEntry_EmptyInput(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var outBuf bytes.Buffer
	// EOF with no input.
	inBuf := bytes.NewBufferString("")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      inBuf,
		OpenURL: func(_ string) error { return fmt.Errorf("no browser") },
	}

	_, err := flow.Run(ctx)
	require.Error(t, err)
}

func TestFlow_Run_CallbackBeatsManualEntry(t *testing.T) {
	// When both callback server and stdin are available, callback wins
	// because it fires first (browser redirect is instant in test).
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"callback-token","token_type":"Bearer"}`)
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

	// Stdin has a code but the callback fires first.
	inBuf := bytes.NewBufferString("stdin-code\n")

	flow := &Flow{
		Config: cfg,
		In:     inBuf,
		OpenURL: func(rawURL string) error {
			go func() {
				parsed, _ := neturl.Parse(rawURL)
				redirectURI := parsed.Query().Get("redirect_uri")
				//nolint:gosec // test-only HTTP request
				resp, err := http.Get(redirectURI + "?code=callback-code")
				if err == nil {
					resp.Body.Close()
				}
			}()
			return nil
		},
	}

	token, err := flow.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "callback-token", token.AccessToken)
}

func TestFlow_Run_ManualCodeEntry_ShowsInstructions_BrowserOK(t *testing.T) {
	// Paste instructions appear even when browser opens OK (remote scenario).
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var outBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      bytes.NewBufferString(""), // EOF, will timeout
		OpenURL: func(_ string) error { return nil }, // browser opens OK
	}

	_, _ = flow.Run(ctx)
	output := outBuf.String()
	assert.Contains(t, output, "Paste URL or code")
	assert.Contains(t, output, "copy the ENTIRE URL")
}

func TestFlow_Run_ManualCodeEntry_ShowsInstructions_BrowserFails(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var outBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      bytes.NewBufferString(""), // EOF, will timeout
		OpenURL: func(_ string) error { return fmt.Errorf("no browser") },
	}

	_, _ = flow.Run(ctx)
	output := outBuf.String()
	assert.Contains(t, output, "Could not open browser")
	assert.Contains(t, output, "Paste URL or code")
}

func TestFlow_Run_ManualCodeEntry_BlankLine(t *testing.T) {
	// User enters just whitespace — should not crash, should timeout.
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var outBuf bytes.Buffer
	inBuf := bytes.NewBufferString("   \n")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	flow := &Flow{
		Config:  cfg,
		Out:     &outBuf,
		In:      inBuf,
		OpenURL: func(_ string) error { return fmt.Errorf("no browser") },
	}

	_, err := flow.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestParseManualInput_BareCode(t *testing.T) {
	ac := parseManualInput("4/abc123")
	assert.Equal(t, "4/abc123", ac.code)
	assert.Empty(t, ac.redirectURI)
}

func TestParseManualInput_FullURL(t *testing.T) {
	ac := parseManualInput("http://127.0.0.1:9999/callback?code=my-code&scope=foo")
	assert.Equal(t, "my-code", ac.code)
	assert.Equal(t, "http://127.0.0.1:9999/callback", ac.redirectURI)
}

func TestParseManualInput_URLWithoutCode(t *testing.T) {
	input := "http://127.0.0.1:9999/callback?state=x"
	ac := parseManualInput(input)
	assert.Equal(t, input, ac.code)
	assert.Empty(t, ac.redirectURI)
}

func TestParseManualInput_InvalidURL(t *testing.T) {
	input := string([]byte{0x7f})
	ac := parseManualInput(input)
	assert.Equal(t, input, ac.code)
	assert.Empty(t, ac.redirectURI)
}

func TestParseManualInput_PreservesPortInRedirectURI(t *testing.T) {
	// Critical: the redirect_uri must include the exact port from the
	// original auth request, not the current callback server's port.
	ac := parseManualInput("http://127.0.0.1:41531/callback?state=state&code=4/0ASc3gC2USmf&scope=foo")
	assert.Equal(t, "4/0ASc3gC2USmf", ac.code)
	assert.Equal(t, "http://127.0.0.1:41531/callback", ac.redirectURI)
}

func TestFlow_Run_NoStdin_NoManualPrompt(t *testing.T) {
	// When In is nil, no manual code entry prompt is shown (original behavior).
	cfg := &oauth2.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://example.com/auth",
			TokenURL: "http://example.com/token",
		},
	}

	var outBuf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	flow := &Flow{
		Config: cfg,
		Out:    &outBuf,
		// In is nil -- no manual code entry.
		OpenURL: func(_ string) error {
			cancel()
			return fmt.Errorf("no browser")
		},
	}

	_, _ = flow.Run(ctx)
	assert.NotContains(t, outBuf.String(), "Paste URL or code")
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
