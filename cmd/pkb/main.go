package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors/gdrive"
	"github.com/cwoolley/personal-knowledge-base/internal/search"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
)

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

	root.AddCommand(searchCmd)
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
	clientID := os.Getenv("PKB_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("PKB_GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return func(_ context.Context, _ string) ([]connectors.Result, error) {
			return nil, fmt.Errorf("Google Drive credentials not configured.\n\n" +
				"Set these environment variables:\n" +
				"  export PKB_GOOGLE_CLIENT_ID=\"your-client-id\"\n" +
				"  export PKB_GOOGLE_CLIENT_SECRET=\"your-client-secret\"\n\n" +
				"See README.md for setup instructions.")
		}
	}

	return func(ctx context.Context, query string) ([]connectors.Result, error) {
		cfg := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{drive.DriveReadonlyScope},
			Endpoint:     google.Endpoint,
		}

		tokenPath := os.Getenv("PKB_TOKEN_PATH")
		if tokenPath == "" {
			tokenPath = "token.json"
		}

		tok, err := gdrive.LoadToken(tokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load OAuth token from %s: %w\n\n"+
				"You may need to complete the OAuth flow first.", tokenPath, err)
		}

		client, err := gdrive.NewAPIClient(ctx, cfg.TokenSource(ctx, tok))
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
