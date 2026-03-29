package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/adrg/xdg"
	authpkg "github.com/maximerivest/mcptocli/internal/auth"
	"github.com/maximerivest/mcptocli/internal/cache"
	"github.com/maximerivest/mcptocli/internal/config"
	"github.com/maximerivest/mcptocli/internal/daemon"
	"github.com/maximerivest/mcptocli/internal/expose"
	mcpclient "github.com/maximerivest/mcptocli/internal/mcp/client"
	"github.com/spf13/cobra"
)

// isOnPath checks whether a directory is in the current PATH.
func isOnPath(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		pAbs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if pAbs == abs {
			return true
		}
	}
	return false
}

func newAddCommand(state *State) *cobra.Command {
	var (
		command   string
		url       string
		auth      string
		bearerEnv string
		local     bool
		roots     []string
		as        string
		noExpose  bool
	)

	cmd := &cobra.Command{
		Use:   "add <name> [command-or-url]",
		Short: "Save a server (local command or remote URL) under a name",
		Long: `Save a server so you can refer to it by name.

The second argument is the command to start a local server, or a URL for a remote one.
URLs (starting with http:// or https://) are detected automatically.`,
		Example: `  # Local server (started on demand)
  mcptocli add time 'uvx mcp-server-time'

  # Remote server with OAuth
  mcptocli add notion https://mcp.notion.com/mcp --auth oauth

  # Remote server with bearer token
  mcptocli add acme https://api.acme.dev/mcp --bearer-env ACME_TOKEN`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := config.NormalizeCommandName(args[0])
			if err != nil {
				return err
			}

			// Second positional: auto-detect URL vs command
			if len(args) == 2 {
				target := strings.TrimSpace(args[1])
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
					if url == "" {
						url = target
					}
				} else {
					if command == "" {
						command = target
					}
				}
			}

			if command == "" && url == "" {
				return fmt.Errorf("usage: mcptocli add <name> <command-or-url>")
			}
			if command != "" && url != "" {
				return fmt.Errorf("--command and --url are mutually exclusive")
			}

			repo, err := state.Repo()
			if err != nil {
				return err
			}

			scope := config.SourceGlobal
			if local {
				scope = config.SourceLocal
			}

			server := &config.Server{
				Command:   strings.TrimSpace(command),
				URL:       strings.TrimSpace(url),
				Auth:      strings.TrimSpace(auth),
				BearerEnv: strings.TrimSpace(bearerEnv),
				Roots:     append([]string(nil), roots...),
			}
			if err := repo.UpsertServer(scope, name, server); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "added server %q\n", name)

			// Clear any stale metadata cache for this server
			clearMetadataCache(state, server)

			// Auto-expose unless opted out
			if !noExpose {
				exposedName := as
				if exposedName == "" {
					exposedName, err = config.DefaultExposeName(name)
					if err != nil {
						return nil // server added, expose is best-effort
					}
				}
				_ = repo.AddExpose(scope, name, exposedName)
				executable, err := os.Executable()
				if err == nil {
					shimPath, err := expose.Create(repo.Paths.ExposeBinDir, exposedName, executable)
					if err == nil {
						fmt.Fprintf(cmd.OutOrStdout(), "exposed as %q (%s)\n", exposedName, shimPath)
						if !isOnPath(repo.Paths.ExposeBinDir) {
							fmt.Fprintf(cmd.ErrOrStderr(), "\nhint: add %s to your PATH to use %q directly\n", repo.Paths.ExposeBinDir, exposedName)
						}
					}
				}
			}

			progName := state.Options.Invocation.ProgramName
			if progName == "" {
				progName = "mcptocli"
			}

			// Auto-login for OAuth servers
			if strings.EqualFold(strings.TrimSpace(auth), "oauth") && strings.TrimSpace(url) != "" {
				store, err := state.TokenStore()
				if err == nil {
					server.Name = name // needed for token key
					ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
					defer cancel()
					fmt.Fprintf(cmd.OutOrStdout(), "\nlogging in...\n")
					token, loginErr := authpkg.LoginOAuth(ctx, store, server)
					if loginErr != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "login failed: %v\n", loginErr)
						fmt.Fprintf(cmd.ErrOrStderr(), "run \"%s %s login\" to try again\n", progName, name)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "login successful (token type: %s)\n", token.TokenType)
						// Fetch and cache tools now that we're authenticated
						refreshMetadataCache(cmd, state, server)
					}
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nnow use it:\n  %s %s tools\n  %s %s shell\n", progName, name, progName, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command (alternative to positional arg)")
	cmd.Flags().StringVar(&url, "url", "", "Remote server URL (alternative to positional arg)")
	cmd.Flags().StringVar(&auth, "auth", "", "Auth mode (e.g. oauth)")
	cmd.Flags().StringVar(&bearerEnv, "bearer-env", "", "Env var containing a bearer token")
	cmd.Flags().StringSliceVar(&roots, "root", nil, "Root path (repeatable)")
	cmd.Flags().BoolVar(&local, "local", false, "Write to .mcptocli.yaml instead of global config")
	cmd.Flags().StringVar(&as, "as", "", "Custom exposed command name")
	cmd.Flags().BoolVar(&noExpose, "no-expose", false, "Skip creating an exposed command")
	return cmd
}

func newListCommand(state *State) *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List saved servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := state.Repo()
			if err != nil {
				return err
			}
			servers, err := repo.ListServers()
			if err != nil {
				return err
			}
			if len(servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no registered servers")
				return nil
			}

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, server := range servers {
				target := server.Command
				if server.URL != "" {
					target = server.URL
				}
				status := ""
				if server.Command != "" && daemon.IsRunning(xdg.DataHome, server.Name) {
					status = " (up)"
				}
				fmt.Fprintf(writer, "%s%s\t%s\n", server.Name, status, target)
			}
			return writer.Flush()
		},
	}
}

func newRemoveCommand(state *State) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a saved server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := state.Repo()
			if err != nil {
				return err
			}
			server, err := repo.ResolveServer(args[0])
			if err != nil {
				return err
			}
			for _, exposedName := range server.ExposeAs {
				if err := expose.Remove(repo.Paths.ExposeBinDir, exposedName); err != nil {
					return err
				}
			}
			clearMetadataCache(state, server)
			if err := repo.RemoveServer(server.Source, server.Name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed server %q from %s config\n", server.Name, server.Source)
			return nil
		},
	}
}

func newExposeCommand(state *State) *cobra.Command {
	var (
		as     string
		remove bool
	)

	cmd := &cobra.Command{
		Use:   "expose <server>",
		Short: "Make a server available as its own command (e.g. mcp-time)",
		Long: `Create (or remove) a standalone command for a server.

After exposing, you can use the server directly without the mcptocli prefix.
For example, exposing "time" creates "mcp-time" so you can run:
  mcp-time tools
  mcp-time get-current-time --timezone UTC`,
		Example: `  # Create mcp-time command
  mcptocli expose time

  # Create with a custom name
  mcptocli expose time --as worldclock

  # Remove the exposed command
  mcptocli expose --remove time`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := state.Repo()
			if err != nil {
				return err
			}
			server, err := repo.ResolveServer(args[0])
			if err != nil {
				return err
			}

			exposedName := strings.TrimSpace(as)
			if exposedName == "" {
				exposedName, err = config.DefaultExposeName(server.Name)
				if err != nil {
					return err
				}
			} else {
				exposedName, err = config.NormalizeCommandName(exposedName)
				if err != nil {
					return err
				}
			}

			if remove {
				if err := repo.RemoveExpose(server.Source, server.Name, exposedName); err != nil {
					return err
				}
				if err := expose.Remove(repo.Paths.ExposeBinDir, exposedName); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "removed exposed command %q for server %q\n", exposedName, server.Name)
				return nil
			}

			if err := repo.AddExpose(server.Source, server.Name, exposedName); err != nil {
				return err
			}
			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("find current executable: %w", err)
			}
			shimPath, err := expose.Create(repo.Paths.ExposeBinDir, exposedName, executable)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "exposed %q as %q\n", server.Name, exposedName)
			fmt.Fprintf(cmd.OutOrStdout(), "shim: %s\n", shimPath)
			fmt.Fprintf(cmd.OutOrStdout(), "add %s to PATH if needed\n", repo.Paths.ExposeBinDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&as, "as", "", "Custom command name (default: mcp-<server>)")
	cmd.Flags().BoolVar(&remove, "remove", false, "Remove the exposed command instead of creating it")
	return cmd
}

// clearMetadataCache removes cached metadata for a server.
func clearMetadataCache(state *State, server *config.Server) {
	store, err := state.MetadataStore()
	if err != nil || store == nil {
		return
	}
	_ = store.Delete(server)
}

// refreshMetadataCache connects to the server and caches its tools.
func refreshMetadataCache(cmd *cobra.Command, state *State, server *config.Server) {
	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	store, err := state.TokenStore()
	if err != nil {
		return
	}
	headers, err := authpkg.HeadersForServer(store, server)
	if err != nil {
		return
	}

	session, err := mcpclient.Connect(ctx, server, headers, mcpclient.ConnectOptions{})
	if err != nil {
		return
	}
	defer session.Close()

	tools, err := session.ListTools(ctx)
	if err != nil {
		return
	}
	resources, _ := session.ListResources(ctx)
	prompts, _ := session.ListPrompts(ctx)

	cacheMetadata(state, server, func(m *cache.Metadata) {
		m.Tools = tools
		m.Resources = resources
		m.Prompts = prompts
	})
}
