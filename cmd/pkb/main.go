package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cwoolley/personal-knowledge-base/internal/apiclient"
	"github.com/cwoolley/personal-knowledge-base/internal/auth"
	pkbweb "github.com/cwoolley/personal-knowledge-base/internal/web"
	"github.com/cwoolley/personal-knowledge-base/internal/config"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gmail"
	"github.com/cwoolley/personal-knowledge-base/internal/search"
	"github.com/cwoolley/personal-knowledge-base/internal/server"
	"github.com/cwoolley/personal-knowledge-base/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
	gm "google.golang.org/api/gmail/v1"
)

// version is set at build time via ldflags: -X main.version=<value>
var version = "dev"

// makeSignalCh creates the OS signal channel. Overridden in tests.
var makeSignalCh = func() (chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch, func() { signal.Stop(ch) }
}

// teaRunner abstracts tea.Program.Run for testability.
type teaRunner interface {
	Run() (tea.Model, error)
}

// newTeaProgram creates a tea.Program. Overridden in tests.
var newTeaProgram = func(model tea.Model) teaRunner {
	return tea.NewProgram(model)
}

// loadConfig loads application config. Overridden in tests.
var loadConfig = config.Load

// newAPIClient creates a Drive API client. Overridden in tests.
var newAPIClient = gdrive.NewAPIClient

// newGmailAPIClient creates a Gmail API client. Overridden in tests.
var newGmailAPIClient = gmail.NewAPIClient

// openBrowser opens a URL in the default browser. Overridden in tests.
var openBrowser = func(rawURL string) error {
	return exec.Command("open", rawURL).Start()
}

// googleOAuthEndpoint returns the Google OAuth2 endpoint. Overridden in tests.
var googleOAuthEndpoint = func() oauth2.Endpoint {
	return google.Endpoint
}

// httpServer abstracts the server for testability of the serve loop.
type httpServer interface {
	Serve() error
	Addr() string
	Shutdown(ctx context.Context) error
}

// SearchFunc abstracts the search operation for testability.
// sources filters which connectors to query; nil means all.
type SearchFunc func(ctx context.Context, query string, sources []string) ([]connectors.Result, error)

func truncateSnippet(s string) string {
	const maxLen = 80
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// searchHandler returns an http.Handler for the /search endpoint.
func searchHandler(searchFn SearchFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required parameter: q"})
			return
		}
		var sources []string
		if s := r.URL.Query().Get("sources"); s != "" {
			sources = strings.Split(s, ",")
		}
		results, err := searchFn(r.Context(), q, sources)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	})
}

// startEmbeddedServer starts a server on :0 with the search handler and
// returns an apiclient pointed at it plus a cleanup function.
var startEmbeddedServer = func(searchFn SearchFunc) (*apiclient.Client, func(), error) {
	srv := server.New(":0")
	srv.Handle("GET /search", searchHandler(searchFn))
	if err := srv.Listen(); err != nil {
		return nil, nil, fmt.Errorf("start embedded server: %w", err)
	}
	go srv.Serve() //nolint:errcheck // shutdown handles cleanup
	baseURL := "http://" + srv.Addr()
	client := apiclient.New(baseURL, http.DefaultClient)
	cleanup := func() { _ = srv.Shutdown(context.Background()) }
	return client, cleanup, nil
}

func newRootCmd(searchFn SearchFunc, out io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "pkb",
		Short: "Personal Knowledge Base — search across all your services",
	}

	searchCmd := &cobra.Command{
		Use:   "search [query...]",
		Short: "Search across all connected services",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cleanup, err := startEmbeddedServer(searchFn)
			if err != nil {
				return err
			}
			defer cleanup()

			query := strings.Join(args, " ")
			results, err := client.Search(cmd.Context(), query, nil)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Fprintln(out, "No results found.")
				return nil
			}

			for i, r := range results {
				if s := truncateSnippet(r.Snippet); s != "" {
					fmt.Fprintf(out, "%d. %s\n   %s\n   %s\n   [%s]\n\n", i+1, r.Title, s, r.URL, r.Source)
				} else {
					fmt.Fprintf(out, "%d. %s\n   %s\n   [%s]\n\n", i+1, r.Title, r.URL, r.Source)
				}
			}
			return nil
		},
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, _ := cmd.Flags().GetString("addr")
			srv := server.New(addr)
			srv.Handle("GET /search", searchHandler(searchFn))
			srv.Handle("GET /", pkbweb.Handler())

			if err := srv.Listen(); err != nil {
				return err
			}
			fmt.Fprintf(out, "Listening on %s\n", srv.Addr())
			return serveLoop(srv, out)
		},
	}
	serveCmd.Flags().String("addr", ":8080", "listen address")

	interactiveCmd := &cobra.Command{
		Use:     "interactive",
		Short:   "Launch the interactive TUI",
		Aliases: []string{"tui"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cleanup, err := startEmbeddedServer(searchFn)
			if err != nil {
				return err
			}
			defer cleanup()

			apiSearch := tui.SearchFunc(client.Search)
			model := tui.NewModel(apiSearch)
			p := newTeaProgram(model)
			_, err = p.Run()
			return err
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of pkb",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(out, "pkb version %s\n", version)
		},
	}

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Google (opens browser for OAuth flow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if appCfg.GoogleClientID == "" || appCfg.GoogleClientSecret == "" {
				return fmt.Errorf("Google credentials not configured.\n\n" +
					"Set these environment variables:\n" +
					"  export PKB_GOOGLE_CLIENT_ID=\"your-client-id\"\n" +
					"  export PKB_GOOGLE_CLIENT_SECRET=\"your-client-secret\"")
			}

			oauthCfg := &oauth2.Config{
				ClientID:     appCfg.GoogleClientID,
				ClientSecret: appCfg.GoogleClientSecret,
				Scopes:       []string{drive.DriveReadonlyScope, gm.GmailReadonlyScope},
				Endpoint:     googleOAuthEndpoint(),
			}

			flow := &auth.Flow{
				Config:  oauthCfg,
				OpenURL: openBrowser,
			}

			fmt.Fprintln(out, "Opening browser for Google authorization...")
			token, err := flow.Run(cmd.Context())
			if err != nil {
				return fmt.Errorf("authorization failed: %w", err)
			}

			if err := gdrive.SaveToken(appCfg.TokenPath, token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}

			fmt.Fprintf(out, "Token saved to %s\n", appCfg.TokenPath)
			return nil
		},
	}

	root.AddCommand(searchCmd)
	root.AddCommand(serveCmd)
	root.AddCommand(interactiveCmd)
	root.AddCommand(versionCmd)
	root.AddCommand(authCmd)
	return root
}

func runWithOutput(args []string, searchFn SearchFunc, out io.Writer) error {
	cmd := newRootCmd(searchFn, out)
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd.Execute()
}

func run(args []string, searchFn SearchFunc) error {
	return runWithOutput(args, searchFn, os.Stdout)
}

func buildSearchFn() SearchFunc {
	appCfg, err := loadConfig()
	if err != nil {
		return func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	if appCfg.GoogleClientID == "" || appCfg.GoogleClientSecret == "" {
		return func(_ context.Context, _ string, _ []string) ([]connectors.Result, error) {
			return nil, fmt.Errorf("Google Drive credentials not configured.\n\n" +
				"Set these environment variables:\n" +
				"  export PKB_GOOGLE_CLIENT_ID=\"your-client-id\"\n" +
				"  export PKB_GOOGLE_CLIENT_SECRET=\"your-client-secret\"\n\n" +
				"See README.md for setup instructions.")
		}
	}

	return func(ctx context.Context, query string, sources []string) ([]connectors.Result, error) {
		oauthCfg := &oauth2.Config{
			ClientID:     appCfg.GoogleClientID,
			ClientSecret: appCfg.GoogleClientSecret,
			Scopes:       []string{drive.DriveReadonlyScope, gm.GmailReadonlyScope},
			Endpoint:     google.Endpoint,
		}

		tok, err := gdrive.LoadToken(appCfg.TokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load OAuth token from %s: %w\n\n"+
				"You may need to complete the OAuth flow first.", appCfg.TokenPath, err)
		}

		client, err := newAPIClient(ctx, oauthCfg.TokenSource(ctx, tok))
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Drive client: %w", err)
		}

		driveConnector := gdrive.NewConnector(client)

		// Create Gmail connector with the same token source.
		gmailClient, err := newGmailAPIClient(ctx, oauthCfg.TokenSource(ctx, tok))
		if err != nil {
			// Gmail is optional — fall back to Drive only.
			engine := search.New(driveConnector)
			return engine.SearchWithSources(ctx, query, sources)
		}
		gmailConnector := gmail.NewConnector(gmailClient)

		engine := search.New(driveConnector, gmailConnector)
		return engine.SearchWithSources(ctx, query, sources)
	}
}

func serveLoop(srv httpServer, out io.Writer) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	sigCh, stopSignals := makeSignalCh()
	defer stopSignals()

	select {
	case sig := <-sigCh:
		fmt.Fprintf(out, "Received %s, shutting down...\n", sig)
		if err := srv.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func main() {
	if err := run(os.Args[1:], buildSearchFn()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
