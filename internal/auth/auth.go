package auth

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

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

	// ListenAddr is the address to listen on for the callback server.
	// Defaults to "127.0.0.1:0" (random port on loopback) if empty.
	ListenAddr string
}

// Run executes the OAuth flow. It blocks until the user completes
// authorization or the context is cancelled.
func (f *Flow) Run(ctx context.Context) (*oauth2.Token, error) {
	codeCh := make(chan string, 1)
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
		codeCh <- code
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

	authURL := f.Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	if err := f.OpenURL(authURL); err != nil {
		out := f.Out
		if out == nil {
			out = os.Stderr
		}
		fmt.Fprintln(out, "Could not open browser automatically.")
		fmt.Fprintln(out, "Open this URL in your browser to authorize:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, authURL)
		fmt.Fprintln(out)
	}

	// Wait for the auth code, an error, or cancellation.
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	token, err := f.Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	return token, nil
}
