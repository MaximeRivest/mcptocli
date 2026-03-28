package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/maximerivest/mcp2cli/internal/auth"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	mcpclient "github.com/maximerivest/mcp2cli/internal/mcp/client"
	"github.com/maximerivest/mcp2cli/internal/serverref"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name   string `json:"name" yaml:"name"`
	Status string `json:"status" yaml:"status"`
	Detail string `json:"detail,omitempty" yaml:"detail,omitempty"`
}

func newDoctorCommand(state *State) *cobra.Command {
	var (
		command           string
		urlValue          string
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
		output            string
	)

	cmd := &cobra.Command{
		Use:   "doctor [server]",
		Short: "Diagnose connection, auth, and config issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}
			explicitServer := ""
			if !state.Options.Invocation.IsExposedCommand() && command == "" && urlValue == "" {
				if len(args) != 1 {
					return exitcode.New(exitcode.Usage, "usage: doctor [server]")
				}
				explicitServer = args[0]
			} else if len(args) > 0 {
				return exitcode.New(exitcode.Usage, "usage: doctor [server]")
			}

			checks := []doctorCheck{}
			renderAndReturn := func(err error) error {
				_ = renderDoctor(cmd.OutOrStdout(), checks, outputMode)
				return err
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
				URL:               urlValue,
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
				checks = append(checks, doctorCheck{Name: "resolve", Status: "fail", Detail: err.Error()})
				return renderAndReturn(err)
			}
			checks = append(checks, doctorCheck{Name: "resolve", Status: "ok", Detail: resolved.DisplayName})

			if resolved.Server.URL != "" {
				if _, err := url.ParseRequestURI(resolved.Server.URL); err != nil {
					checks = append(checks, doctorCheck{Name: "url", Status: "fail", Detail: err.Error()})
					return renderAndReturn(exitcode.Wrap(exitcode.Config, err, "parse server URL"))
				}
				checks = append(checks, doctorCheck{Name: "url", Status: "ok", Detail: "valid remote URL"})
			} else {
				argv, err := shellquote.Split(resolved.Server.Command)
				if err != nil {
					checks = append(checks, doctorCheck{Name: "command", Status: "fail", Detail: err.Error()})
					return renderAndReturn(exitcode.Wrap(exitcode.Usage, err, "parse command"))
				}
				if len(argv) == 0 {
					checks = append(checks, doctorCheck{Name: "command", Status: "fail", Detail: "empty command"})
					return renderAndReturn(exitcode.New(exitcode.Config, "command cannot be empty"))
				}
				path, err := exec.LookPath(argv[0])
				if err != nil {
					checks = append(checks, doctorCheck{Name: "command", Status: "fail", Detail: err.Error()})
					return renderAndReturn(exitcode.Wrapf(exitcode.Config, err, "find executable %q", argv[0]))
				}
				checks = append(checks, doctorCheck{Name: "command", Status: "ok", Detail: path})
			}

			store, err := state.TokenStore()
			if err != nil {
				return err
			}
			resolvedHeaders, err := auth.HeadersForServer(store, resolved.Server)
			if err != nil {
				checks = append(checks, doctorCheck{Name: "auth", Status: "fail", Detail: err.Error()})
				return renderAndReturn(err)
			}
			if len(resolvedHeaders) > 0 {
				checks = append(checks, doctorCheck{Name: "auth", Status: "ok", Detail: fmt.Sprintf("%d header(s) prepared", len(resolvedHeaders))})
			} else {
				checks = append(checks, doctorCheck{Name: "auth", Status: "ok", Detail: "no auth required"})
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			session, err := mcpclient.Connect(ctx, resolved.Server, resolvedHeaders)
			if err != nil {
				checks = append(checks, doctorCheck{Name: "connect", Status: "fail", Detail: err.Error()})
				return renderAndReturn(err)
			}
			defer func() { _ = session.Close() }()
			checks = append(checks, doctorCheck{Name: "connect", Status: "ok", Detail: "initialize handshake succeeded"})

			tools, err := session.ListTools(ctx)
			if err != nil {
				checks = append(checks, doctorCheck{Name: "tools", Status: "fail", Detail: err.Error()})
				return renderAndReturn(err)
			}
			checks = append(checks, doctorCheck{Name: "tools", Status: "ok", Detail: fmt.Sprintf("%d tool(s) available", len(tools))})

			return renderDoctor(cmd.OutOrStdout(), checks, outputMode)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(&urlValue, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "Additional HTTP header in Key: Value form (repeatable)")
	cmd.Flags().StringVar(&authMode, "auth", "", "Authentication mode override (oauth or bearer)")
	cmd.Flags().StringVar(&bearerEnv, "bearer-env", "", "Environment variable containing a bearer token")
	cmd.Flags().StringVar(&oauthAuthorizeURL, "oauth-authorize-url", "", "OAuth authorize URL override")
	cmd.Flags().StringVar(&oauthTokenURL, "oauth-token-url", "", "OAuth token URL override")
	cmd.Flags().StringVar(&oauthClientID, "oauth-client-id", "", "OAuth client ID override")
	cmd.Flags().StringSliceVar(&oauthScopes, "oauth-scope", nil, "OAuth scope (repeatable)")
	cmd.Flags().DurationVar(&timeout, "timeout", mcpclient.DefaultTimeout(), "Doctor timeout")
	cmd.Flags().StringVarP(&output, "output", "o", "auto", "Output format: auto, json, or yaml")
	return cmd
}

func renderDoctor(w io.Writer, checks []doctorCheck, output string) error {
	switch output {
	case "auto", "table":
		writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "CHECK\tSTATUS\tDETAIL")
		for _, check := range checks {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", check.Name, check.Status, strings.TrimSpace(check.Detail))
		}
		return writer.Flush()
	case "json":
		payload, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(append(payload, '\n'))
		return err
	case "yaml":
		return writeYAML(w, checks)
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}
