package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/maximerivest/mcp2cli/internal/auth"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/maximerivest/mcp2cli/internal/serverref"
	"github.com/spf13/cobra"
)

func newLoginCommand(state *State) *cobra.Command {
	var (
		command           string
		url               string
		cwd               string
		envVars           []string
		headers           []string
		authMode          string
		bearerEnv         string
		oauthAuthorizeURL string
		oauthTokenURL     string
		oauthClientID     string
		oauthScopes       []string
		timeout           time.Duration
	)

	cmd := &cobra.Command{
		Use:   "login [server]",
		Short: "Authenticate with a remote server",
		Long: `Trigger the OAuth or bearer authentication flow for a remote server.

Useful for pre-authenticating before using tools, or refreshing tokens.`,
		Example: `  # Login to an OAuth server
  mcp2cli login notion

  # Verify bearer token is available
  mcp2cli login acme`,
		RunE: func(cmd *cobra.Command, args []string) error {
			explicitServer := ""
			if !state.Options.Invocation.IsExposedCommand() && command == "" && url == "" {
				if len(args) != 1 {
					return exitcode.New(exitcode.Usage, "usage: login [server]")
				}
				explicitServer = args[0]
			} else if len(args) > 0 {
				return exitcode.New(exitcode.Usage, "usage: login [server]")
			}

			repo, err := state.Repo()
			if err != nil {
				return err
			}
			bound, err := state.BoundServer()
			if err != nil {
				return err
			}
			resolved, err := serverref.Resolve(repo, bound, serverref.Options{
				ExplicitName:      explicitServer,
				Command:           command,
				URL:               url,
				CWD:               cwd,
				Env:               envVars,
				Headers:           headers,
				Auth:              authMode,
				BearerEnv:         bearerEnv,
				OAuthAuthorizeURL: oauthAuthorizeURL,
				OAuthTokenURL:     oauthTokenURL,
				OAuthClientID:     oauthClientID,
				OAuthScopes:       oauthScopes,
			})
			if err != nil {
				return err
			}
			if resolved.Server.URL == "" {
				return exitcode.New(exitcode.Config, "login requires a remote server URL")
			}

			switch resolved.Server.Auth {
			case "oauth":
				store, err := state.TokenStore()
				if err != nil {
					return err
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
				token, err := auth.LoginOAuth(ctx, store, resolved.Server)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "login successful for %q\n", resolved.DisplayName)
				fmt.Fprintf(cmd.OutOrStdout(), "token type: %s\n", token.TokenType)
				return nil
			case "bearer":
				if resolved.Server.BearerEnv == "" {
					return exitcode.New(exitcode.Auth, "bearer auth requires --bearer-env or bearer_env config")
				}
				if _, err := auth.HeadersForServer(nil, resolved.Server); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "bearer token available via %s\n", resolved.Server.BearerEnv)
				return nil
			default:
				return exitcode.New(exitcode.Auth, "login only supports servers configured with auth: oauth or auth: bearer")
			}
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(&url, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "Additional HTTP header in Key: Value form (repeatable)")
	cmd.Flags().StringVar(&authMode, "auth", "", "Authentication mode override (oauth or bearer)")
	cmd.Flags().StringVar(&bearerEnv, "bearer-env", "", "Environment variable containing a bearer token")
	cmd.Flags().StringVar(&oauthAuthorizeURL, "oauth-authorize-url", "", "OAuth authorize URL override")
	cmd.Flags().StringVar(&oauthTokenURL, "oauth-token-url", "", "OAuth token URL override")
	cmd.Flags().StringVar(&oauthClientID, "oauth-client-id", "", "OAuth client ID override")
	cmd.Flags().StringSliceVar(&oauthScopes, "oauth-scope", nil, "OAuth scope (repeatable)")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Login timeout")

	for _, name := range []string{"command", "url", "cwd", "env", "header", "auth", "bearer-env", "oauth-authorize-url", "oauth-token-url", "oauth-client-id", "oauth-scope"} {
		markConnectionFlag(cmd, name)
	}
	useGroupedHelp(cmd)
	return cmd
}
