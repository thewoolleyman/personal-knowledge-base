package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

// BrowserOpener is a function that opens a URL in the default browser.
// Injected for testability.
type BrowserOpener func(url string) error

// Flow performs an interactive OAuth2 authorization code flow.
// It starts a local HTTP server on a random port, directs the user's
// browser to the authorization URL, waits for the callback with the
// auth code, exchanges it for a token, and returns the token.
type Flow struct {
	Config  *oauth2.Config
	OpenURL BrowserOpener

	// Out is where fallback messages (e.g. manual URL) are written.
	// Defaults to os.Stderr if nil.
	Out io.Writer

	// In is where manual code entry is read from (typically os.Stdin).
	// When non-nil and the browser cannot be opened, the user is prompted
	// to paste the authorization code or callback URL. When nil, only
	// the callback server path is available.
	In io.Reader

	// ListenAddr is the address to listen on for the callback server.
	// Defaults to "127.0.0.1:0" (random port on loopback) if empty.
	ListenAddr string
}

// authCode carries a code and optional redirect URI extracted from manual input.
type authCode struct {
	code        string
	redirectURI string // non-empty when extracted from a pasted URL
}

// Run executes the OAuth flow. It blocks until the user completes
// authorization or the context is cancelled.
func (f *Flow) Run(ctx context.Context) (*oauth2.Token, error) {
	codeCh := make(chan authCode, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "Authorization failed: no code received", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "Authorization successful! You can close this tab.")
		codeCh <- authCode{code: code}
	})

	listenAddr := f.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1:0"
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Shutdown(ctx) }()

	// Point the redirect URL to the local callback server.
	f.Config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port)

	authURL := f.Config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	out := f.Out
	if out == nil {
		out = os.Stderr
	}
	if err := f.OpenURL(authURL); err != nil {
		fmt.Fprintln(out, "Could not open browser automatically.")
		fmt.Fprintln(out, "Open this URL in your browser to authorize:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, authURL)
		fmt.Fprintln(out)
	}

	// When In is set, always offer manual code entry as a fallback.
	// This handles the common case where the browser opens on a
	// different machine (e.g. SSH session) and the localhost redirect
	// is not reachable.
	if f.In != nil {
		fmt.Fprintln(out, "After authorizing, the browser will redirect to a localhost URL.")
		fmt.Fprintln(out, "If the page fails to load, copy the ENTIRE URL from your browser's address bar")
		fmt.Fprintln(out, "and paste it below:")
		fmt.Fprintln(out)
		fmt.Fprint(out, "Paste URL or code: ")
		go f.readManualCode(codeCh)
	}

	// Wait for the auth code, an error, or cancellation.
	var ac authCode
	select {
	case ac = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// When the code was pasted from a URL, the redirect_uri in the
	// exchange must match the one Google used during authorization
	// (which may be on a different port than the current callback server).
	if ac.redirectURI != "" {
		f.Config.RedirectURL = ac.redirectURI
	}

	token, err := f.Config.Exchange(ctx, ac.code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	return token, nil
}

// readManualCode reads a line from f.In and sends it to codeCh.
// The input may be a bare authorization code or a full callback URL
// containing a code= query parameter.
func (f *Flow) readManualCode(codeCh chan<- authCode) {
	scanner := bufio.NewScanner(f.In)
	if !scanner.Scan() {
		// EOF or read error — do nothing; the select in Run will
		// either get a callback or be cancelled by the context.
		return
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return
	}
	codeCh <- parseManualInput(line)
}

// parseManualInput extracts a code (and optionally the redirect URI)
// from either a bare authorization code or a full callback URL.
// When the input is a URL with code= and a path, the redirect URI
// (scheme + host + path) is returned so the token exchange uses the
// same redirect_uri that Google issued the code for.
func parseManualInput(input string) authCode {
	parsed, err := neturl.Parse(input)
	if err != nil {
		return authCode{code: input}
	}
	code := parsed.Query().Get("code")
	if code == "" {
		return authCode{code: input}
	}
	// Reconstruct the redirect_uri (scheme + host + path, no query).
	redirectURI := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path)
	return authCode{code: code, redirectURI: redirectURI}
}
