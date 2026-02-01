package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cwoolley/personal-knowledge-base/internal/config"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	"github.com/cwoolley/personal-knowledge-base/internal/search"
	"github.com/cwoolley/personal-knowledge-base/internal/server"
	"github.com/cwoolley/personal-knowledge-base/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
)

// makeSignalCh creates the OS signal channel. Overridden in tests.
var makeSignalCh = func() (chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch, func() { signal.Stop(ch) }
}

// SearchFunc abstracts the search operation for testability.
type SearchFunc func(ctx context.Context, query string) ([]connectors.Result, error)

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
			query := strings.Join(args, " ")
			results, err := searchFn(cmd.Context(), query)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Fprintln(out, "No results found.")
				return nil
			}

			for i, r := range results {
				fmt.Fprintf(out, "%d. %s\n   %s\n   [%s]\n\n", i+1, r.Title, r.URL, r.Source)
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
			if err := srv.Listen(); err != nil {
				return err
			}
			fmt.Fprintf(out, "Listening on %s\n", srv.Addr())

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
		},
	}
	serveCmd.Flags().String("addr", ":8080", "listen address")

	interactiveCmd := &cobra.Command{
		Use:     "interactive",
		Short:   "Launch the interactive TUI",
		Aliases: []string{"tui"},
		RunE: func(cmd *cobra.Command, args []string) error {
			model := tui.NewModel(tui.SearchFunc(searchFn))
			p := tea.NewProgram(model)
			_, err := p.Run()
			return err
		},
	}

	root.AddCommand(searchCmd)
	root.AddCommand(serveCmd)
	root.AddCommand(interactiveCmd)
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
	appCfg, err := config.Load()
	if err != nil {
		return func(_ context.Context, _ string) ([]connectors.Result, error) {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	if appCfg.GoogleClientID == "" || appCfg.GoogleClientSecret == "" {
		return func(_ context.Context, _ string) ([]connectors.Result, error) {
			return nil, fmt.Errorf("Google Drive credentials not configured.\n\n" +
				"Set these environment variables:\n" +
				"  export PKB_GOOGLE_CLIENT_ID=\"your-client-id\"\n" +
				"  export PKB_GOOGLE_CLIENT_SECRET=\"your-client-secret\"\n\n" +
				"See README.md for setup instructions.")
		}
	}

	return func(ctx context.Context, query string) ([]connectors.Result, error) {
		oauthCfg := &oauth2.Config{
			ClientID:     appCfg.GoogleClientID,
			ClientSecret: appCfg.GoogleClientSecret,
			Scopes:       []string{drive.DriveReadonlyScope},
			Endpoint:     google.Endpoint,
		}

		tok, err := gdrive.LoadToken(appCfg.TokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load OAuth token from %s: %w\n\n"+
				"You may need to complete the OAuth flow first.", appCfg.TokenPath, err)
		}

		client, err := gdrive.NewAPIClient(ctx, oauthCfg.TokenSource(ctx, tok))
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Drive client: %w", err)
		}

		connector := gdrive.NewConnector(client)
		engine := search.New(connector)
		return engine.Search(ctx, query)
	}
}

func main() {
	if err := run(os.Args[1:], buildSearchFn()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
