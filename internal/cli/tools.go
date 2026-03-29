package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/maximerivest/mcp2cli/internal/auth"
	"github.com/maximerivest/mcp2cli/internal/cache"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/maximerivest/mcp2cli/internal/invoke"
	mcpclient "github.com/maximerivest/mcp2cli/internal/mcp/client"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
	"github.com/maximerivest/mcp2cli/internal/naming"
	"github.com/maximerivest/mcp2cli/internal/schema/inspect"
	"github.com/maximerivest/mcp2cli/internal/serverref"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newToolsCommand(state *State) *cobra.Command {
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
		output            string
	)

	cmd := &cobra.Command{
		Use:   "tools [server] [tool]",
		Short: "List tools or inspect one tool",
		Long: `List all tools on a server, or inspect a single tool's schema and usage.

Without a tool name, shows a table of all available tools.
With a tool name, shows the full usage, arguments, and types.`,
		Example: `  # List all tools on a registered server
  mcp2cli tools weather

  # Inspect a specific tool
  mcp2cli tools weather get-forecast

  # List tools on a one-off server
  mcp2cli tools --url https://mcp.example.com/sse

  # Output as JSON
  mcp2cli tools weather -o json`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return metadataToolsCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}

			explicitServer, toolName, err := parseToolsArgs(state, args, command, url)
			if err != nil {
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

			store, err := state.TokenStore()
			if err != nil {
				return err
			}
			resolvedHeaders, err := auth.HeadersForServer(store, resolved.Server)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			session, err := mcpclient.Connect(ctx, resolved.Server, resolvedHeaders, terminalConnectOptions(cmd.InOrStdin(), cmd.ErrOrStderr()))
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()

			tools, err := session.ListTools(ctx)
			if err != nil {
				return err
			}
			cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Tools = tools })
			sort.Slice(tools, func(i, j int) bool { return naming.ToKebabCase(tools[i].Name) < naming.ToKebabCase(tools[j].Name) })

			if toolName != "" {
				tool, ok := findTool(tools, toolName)
				if !ok {
					return exitcode.WithHint(exitcode.Newf(exitcode.Usage, "tool %q not found", toolName), toolsInspectHint(state, resolved, toolName))
				}
				spec, err := inspect.InspectTool(tool)
				if err != nil {
					return err
				}
				usagePrefix := toolUsagePrefix(state, resolved)
				return renderTool(cmd.OutOrStdout(), tool, spec, usagePrefix, outputMode)
			}
			hint := toolsListHints(state, resolved)
			return renderTools(cmd.OutOrStdout(), tools, outputMode, renderToolsOptions{Hint: hint})
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(&url, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "Additional HTTP header in Key: Value form (repeatable)")
	cmd.Flags().StringVar(&authMode, "auth", "", "Authentication mode override (e.g. oauth)")
	cmd.Flags().StringVar(&bearerEnv, "bearer-env", "", "Environment variable containing a bearer token")
	cmd.Flags().StringVar(&oauthAuthorizeURL, "oauth-authorize-url", "", "OAuth authorize URL override")
	cmd.Flags().StringVar(&oauthTokenURL, "oauth-token-url", "", "OAuth token URL override")
	cmd.Flags().StringVar(&oauthClientID, "oauth-client-id", "", "OAuth client ID override")
	cmd.Flags().StringSliceVar(&oauthScopes, "oauth-scope", nil, "OAuth scope (repeatable)")
	cmd.Flags().DurationVar(&timeout, "timeout", mcpclient.DefaultTimeout(), "Request timeout")
	cmd.Flags().StringVarP(&output, "output", "o", "auto", "Output format: auto, json, yaml, raw, or table")

	for _, name := range []string{"command", "url", "cwd", "env", "header", "auth", "bearer-env", "oauth-authorize-url", "oauth-token-url", "oauth-client-id", "oauth-scope"} {
		markConnectionFlag(cmd, name)
	}
	useGroupedHelp(cmd)
	return cmd
}

func newToolCommand(state *State) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "tool [server] <tool> [args...]",
		Short:              "Invoke a tool",
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return toolCommandCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseToolInvocationTokens(state, args)
			if err != nil {
				return err
			}
			if parsed.Help {
				return cmd.Help()
			}
			outputMode, err := normalizeOutputMode(parsed.Output)
			if err != nil {
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
				ExplicitName:      parsed.ServerName,
				Command:           parsed.Command,
				URL:               parsed.URL,
				CWD:               parsed.CWD,
				Env:               parsed.Env,
				Headers:           parsed.Headers,
				Auth:              parsed.Auth,
				BearerEnv:         parsed.BearerEnv,
				OAuthAuthorizeURL: parsed.OAuthAuthorizeURL,
				OAuthTokenURL:     parsed.OAuthTokenURL,
				OAuthClientID:     parsed.OAuthClientID,
				OAuthScopes:       parsed.OAuthScopes,
			})
			if err != nil {
				return err
			}

			store, err := state.TokenStore()
			if err != nil {
				return err
			}
			resolvedHeaders, err := auth.HeadersForServer(store, resolved.Server)
			if err != nil {
				return err
			}

			// Try cached schema first to skip the tools/list round trip
			var tool types.Tool
			var toolFound bool
			var arguments map[string]any

			if cached := cachedToolSchema(state, resolved.Server, parsed.ToolName); cached != nil {
				tool = *cached
				toolFound = true
			}

			if toolFound {
				spec, err := inspect.InspectTool(tool)
				if err != nil {
					return err
				}
				if parsed.Input != "" {
					if len(parsed.DynamicArgs) > 0 {
						return exitcode.New(exitcode.Usage, "cannot combine --input with schema-derived tool arguments")
					}
					arguments, err = loadInputArguments(parsed.Input)
				} else {
					arguments, err = invoke.ParseToolArguments(spec, parsed.DynamicArgs)
				}
				if err != nil {
					return exitcode.WithHint(err, toolsInspectHint(state, resolved, parsed.ToolName))
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), parsed.Timeout)
			defer cancel()

			session, err := mcpclient.Connect(ctx, resolved.Server, resolvedHeaders, terminalConnectOptions(cmd.InOrStdin(), cmd.ErrOrStderr()))
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()

			// If no cache hit, fall back to listing tools from the server
			if !toolFound {
				tools, err := session.ListTools(ctx)
				if err != nil {
					return err
				}
				cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Tools = tools })
				tool, toolFound = findTool(tools, parsed.ToolName)
				if !toolFound {
					return exitcode.WithHint(exitcode.Newf(exitcode.Usage, "tool %q not found", parsed.ToolName), toolsListHint(state, resolved))
				}
				spec, err := inspect.InspectTool(tool)
				if err != nil {
					return err
				}
				if parsed.Input != "" {
					if len(parsed.DynamicArgs) > 0 {
						return exitcode.New(exitcode.Usage, "cannot combine --input with schema-derived tool arguments")
					}
					arguments, err = loadInputArguments(parsed.Input)
				} else {
					arguments, err = invoke.ParseToolArguments(spec, parsed.DynamicArgs)
				}
				if err != nil {
					return exitcode.WithHint(err, toolsInspectHint(state, resolved, parsed.ToolName))
				}
			}

			result, err := session.CallTool(ctx, tool.Name, arguments)
			if err != nil {
				return err
			}
			return renderToolResult(cmd.OutOrStdout(), result, outputMode)
		},
	}
	return cmd
}

type parsedToolInvocation struct {
	ServerName        string
	ToolName          string
	Command           string
	URL               string
	CWD               string
	Env               []string
	Headers           []string
	Auth              string
	BearerEnv         string
	OAuthAuthorizeURL string
	OAuthTokenURL     string
	OAuthClientID     string
	OAuthScopes       []string
	Input             string
	Output            string
	Timeout           time.Duration
	DynamicArgs       []string
	Help              bool
}

func parseToolInvocationTokens(state *State, tokens []string) (*parsedToolInvocation, error) {
	parsed := &parsedToolInvocation{Output: "auto", Timeout: mcpclient.DefaultTimeout()}

	remaining := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == "-h" || token == "--help" {
			parsed.Help = true
			continue
		}

		name, value, recognized, err := parseKnownToolFlag(tokens, &i)
		if err != nil {
			return nil, err
		}
		if !recognized {
			remaining = append(remaining, token)
			continue
		}

		switch name {
		case "command":
			parsed.Command = value
		case "url":
			parsed.URL = value
		case "cwd":
			parsed.CWD = value
		case "env":
			parsed.Env = append(parsed.Env, value)
		case "header":
			parsed.Headers = append(parsed.Headers, value)
		case "auth":
			parsed.Auth = value
		case "bearer-env":
			parsed.BearerEnv = value
		case "oauth-authorize-url":
			parsed.OAuthAuthorizeURL = value
		case "oauth-token-url":
			parsed.OAuthTokenURL = value
		case "oauth-client-id":
			parsed.OAuthClientID = value
		case "oauth-scope":
			parsed.OAuthScopes = append(parsed.OAuthScopes, value)
		case "input":
			parsed.Input = value
		case "output":
			parsed.Output = value
		case "timeout":
			parsed.Timeout, err = time.ParseDuration(value)
			if err != nil {
				return nil, exitcode.Wrap(exitcode.Usage, err, "parse --timeout")
			}
		}
	}

	if parsed.Help {
		return parsed, nil
	}

	directMode := state.Options.Invocation.IsExposedCommand() || parsed.Command != "" || parsed.URL != ""
	if directMode {
		if len(remaining) < 1 {
			return nil, exitcode.New(exitcode.Usage, "usage: tool [server] <tool> [args...]")
		}
		parsed.ToolName = remaining[0]
		parsed.DynamicArgs = append([]string(nil), remaining[1:]...)
		return parsed, nil
	}
	if len(remaining) < 2 {
		return nil, exitcode.New(exitcode.Usage, "usage: tool [server] <tool> [args...]")
	}
	parsed.ServerName = remaining[0]
	parsed.ToolName = remaining[1]
	parsed.DynamicArgs = append([]string(nil), remaining[2:]...)
	return parsed, nil
}

func parseKnownToolFlag(tokens []string, index *int) (name, value string, recognized bool, err error) {
	token := tokens[*index]
	if token == "-o" {
		if *index+1 >= len(tokens) {
			return "", "", true, exitcode.New(exitcode.Usage, "missing value for -o")
		}
		*index = *index + 1
		return "output", tokens[*index], true, nil
	}
	if !strings.HasPrefix(token, "--") {
		return "", "", false, nil
	}

	flag := strings.TrimPrefix(token, "--")
	if before, after, ok := strings.Cut(flag, "="); ok {
		if isKnownToolFlag(before) {
			return before, after, true, nil
		}
		return "", "", false, nil
	}
	if !isKnownToolFlag(flag) {
		return "", "", false, nil
	}
	if *index+1 >= len(tokens) {
		return "", "", true, exitcode.Newf(exitcode.Usage, "missing value for --%s", flag)
	}
	*index = *index + 1
	return flag, tokens[*index], true, nil
}

func isKnownToolFlag(flag string) bool {
	switch flag {
	case "command", "url", "cwd", "env", "header", "auth", "bearer-env", "oauth-authorize-url", "oauth-token-url", "oauth-client-id", "oauth-scope", "input", "output", "timeout":
		return true
	default:
		return false
	}
}

func parseToolsArgs(state *State, args []string, command string, url string) (string, string, error) {
	if state.Options.Invocation.IsExposedCommand() || command != "" || url != "" {
		switch len(args) {
		case 0:
			return "", "", nil
		case 1:
			return "", args[0], nil
		default:
			return "", "", exitcode.New(exitcode.Usage, "usage: tools [server] [tool]")
		}
	}

	switch len(args) {
	case 1:
		return args[0], "", nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", exitcode.New(exitcode.Usage, "usage: tools [server] [tool]")
	}
}

type renderToolsOptions struct {
	// Hint is printed after the table in auto/table mode (e.g. "Use ... to inspect a tool").
	Hint string
}

func renderTools(w io.Writer, tools []types.Tool, output string, opts ...renderToolsOptions) error {
	views := make([]toolView, 0, len(tools))
	for _, tool := range tools {
		views = append(views, newToolView(tool))
	}

	var hint string
	if len(opts) > 0 {
		hint = opts[0].Hint
	}

	switch output {
	case "json":
		return writeJSON(w, views)
	case "yaml":
		return writeYAML(w, views)
	case "raw":
		for _, view := range views {
			fmt.Fprintln(w, view.Name)
		}
		return nil
	case "table", "auto":
		fmt.Fprintf(w, "Tools (%d):\n\n", len(views))
		writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, view := range views {
			fmt.Fprintf(writer, "  %s\t%s\n", view.Name, truncateDescription(view.Description))
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		if hint != "" {
			fmt.Fprintf(w, "\n%s\n", hint)
		}
		return nil
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

// truncateDescription returns the first sentence of a description,
// capped at 60 characters.
func truncateDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	// Take first sentence (up to first ". " or ".\n" boundary).
	if idx := strings.Index(desc, ". "); idx >= 0 {
		desc = desc[:idx+1]
	} else if idx := strings.Index(desc, ".\n"); idx >= 0 {
		desc = desc[:idx+1]
	}
	const maxLen = 60
	if len(desc) > maxLen {
		return desc[:maxLen-3] + "..."
	}
	return desc
}

func renderTool(w io.Writer, tool types.Tool, spec *inspect.ToolSpec, usagePrefix, output string) error {
	view := newToolView(tool)
	view.Args = make([]toolArgView, 0, len(spec.Arguments))
	for _, arg := range spec.Arguments {
		view.Args = append(view.Args, toolArgView{
			Name:        arg.CLIName,
			Type:        inspect.Placeholder(arg),
			Description: arg.Description,
			Required:    arg.Required,
			Default:     arg.Default,
			HasDefault:  arg.HasDefault,
		})
	}
	if len(spec.RawSchema) > 0 {
		_ = json.Unmarshal(spec.RawSchema, &view.InputSchema)
	}

	switch output {
	case "json":
		return writeJSON(w, view)
	case "yaml":
		return writeYAML(w, view)
	case "raw":
		if len(spec.RawSchema) == 0 {
			_, err := fmt.Fprintln(w, tool.Name)
			return err
		}
		_, err := w.Write(append(spec.RawSchema, '\n'))
		return err
	case "table", "auto":
		fmt.Fprintf(w, "NAME\n  %s", view.Name)
		if view.Description != "" {
			fmt.Fprintf(w, " - %s", view.Description)
		}
		fmt.Fprintln(w)
		if len(spec.Arguments) > 0 {
			fmt.Fprintf(w, "\nUSAGE\n  %s %s", usagePrefix, view.Name)
			for _, part := range spec.UsageParts() {
				fmt.Fprintf(w, " %s", part)
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w, "\nARGS")
			writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			for _, arg := range view.Args {
				details := arg.Description
				if arg.Required {
					details = "Required. " + details
				}
				if arg.HasDefault {
					if details != "" {
						details += " "
					}
					details += fmt.Sprintf("Default: %v.", arg.Default)
				}
				fmt.Fprintf(writer, "  --%s %s\t%s\n", arg.Name, arg.Type, strings.TrimSpace(details))
			}
			if err := writer.Flush(); err != nil {
				return err
			}
		}
		return nil
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func renderToolResult(w io.Writer, result *types.CallToolResult, output string) error {
	switch output {
	case "json":
		return writeJSON(w, result)
	case "yaml":
		return writeYAML(w, result)
	case "raw":
		text, ok := textOnlyContent(result)
		if !ok {
			return exitcode.New(exitcode.Usage, "-o raw is only supported for text tool results")
		}
		_, err := io.WriteString(w, text)
		if err == nil && !strings.HasSuffix(text, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case "table":
		if renderStructuredTable(w, result.StructuredContent) {
			return nil
		}
		fallthrough
	case "auto":
		if text, ok := textOnlyContent(result); ok {
			_, err := io.WriteString(w, text)
			if err == nil && !strings.HasSuffix(text, "\n") {
				_, err = io.WriteString(w, "\n")
			}
			return err
		}
		if len(result.StructuredContent) > 0 {
			return writeJSON(w, result.StructuredContent)
		}
		return writeJSON(w, result)
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func normalizeOutputMode(output string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "auto":
		return "auto", nil
	case "json", "yaml", "raw", "table":
		return strings.ToLower(strings.TrimSpace(output)), nil
	default:
		return "", exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func toolUsagePrefix(state *State, resolved *serverref.Resolved) string {
	if state.Options.Invocation.IsExposedCommand() {
		return state.Options.Invocation.ExposedCommandName
	}
	if resolved != nil && resolved.Server != nil {
		switch {
		case resolved.Server.Source == "ephemeral" && resolved.Server.Command != "":
			return fmt.Sprintf("mcp2cli tool --command %s", resolved.Server.Command)
		case resolved.Server.Source == "ephemeral" && resolved.Server.URL != "":
			return fmt.Sprintf("mcp2cli tool --url %s", resolved.Server.URL)
		case resolved.Server.Name != "":
			return fmt.Sprintf("mcp2cli tool %s", resolved.Server.Name)
		}
	}
	return "mcp2cli tool"
}

// toolsListHints returns the footer hints for the tools listing.
func toolsListHints(state *State, resolved *serverref.Resolved) string {
	if state.Options.Invocation.IsExposedCommand() {
		name := state.Options.Invocation.ExposedCommandName
		return fmt.Sprintf("Inspect:  %s tools <tool>\nInvoke:   %s <tool> [args...]", name, name)
	}
	if resolved != nil && resolved.Server != nil {
		switch {
		case resolved.Server.Source == "ephemeral" && resolved.Server.Command != "":
			return fmt.Sprintf("Inspect:  mcp2cli tools --command %s <tool>\nInvoke:   mcp2cli tool --command %s <tool> [args...]", resolved.Server.Command, resolved.Server.Command)
		case resolved.Server.Source == "ephemeral" && resolved.Server.URL != "":
			return fmt.Sprintf("Inspect:  mcp2cli tools --url %s <tool>\nInvoke:   mcp2cli tool --url %s <tool> [args...]", resolved.Server.URL, resolved.Server.URL)
		case resolved.Server.Name != "":
			return fmt.Sprintf("Inspect:  mcp2cli tools %s <tool>\nInvoke:   mcp2cli %s <tool> [args...]", resolved.Server.Name, resolved.Server.Name)
		}
	}
	return ""
}

func findTool(tools []types.Tool, requested string) (types.Tool, bool) {
	requestedCLI := naming.ToKebabCase(requested)
	for _, tool := range tools {
		if tool.Name == requested || naming.ToKebabCase(tool.Name) == requestedCLI {
			return tool, true
		}
	}
	return types.Tool{}, false
}

func loadInputArguments(input string) (map[string]any, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return map[string]any{}, nil
	}

	data, err := readInputBytes(input)
	if err != nil {
		return nil, err
	}
	arguments := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return arguments, nil
	}
	if err := json.Unmarshal(data, &arguments); err != nil {
		return nil, exitcode.Wrap(exitcode.Usage, err, "parse --input as JSON object")
	}
	return arguments, nil
}

func readInputBytes(input string) ([]byte, error) {
	if strings.HasPrefix(input, "@") {
		return invoke.ReadAtValue(input)
	}
	return []byte(input), nil
}

func writeJSON(w io.Writer, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(payload, '\n'))
	return err
}

func writeYAML(w io.Writer, value any) error {
	payload, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func textOnlyContent(result *types.CallToolResult) (string, bool) {
	if result == nil || len(result.Content) == 0 {
		return "", false
	}
	texts := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		if item.Type != "text" {
			return "", false
		}
		texts = append(texts, item.Text)
	}
	return strings.Join(texts, "\n"), true
}

func renderStructuredTable(w io.Writer, structured map[string]any) bool {
	itemsRaw, ok := structured["items"]
	if !ok {
		return false
	}
	items, ok := itemsRaw.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			return false
		}
		rows = append(rows, row)
	}
	columns := make([]string, 0, len(rows[0]))
	for key := range rows[0] {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, strings.Join(columns, "\t"))
	for _, row := range rows {
		parts := make([]string, 0, len(columns))
		for _, column := range columns {
			parts = append(parts, fmt.Sprint(row[column]))
		}
		fmt.Fprintln(writer, strings.Join(parts, "\t"))
	}
	_ = writer.Flush()
	return true
}

func toolsListHint(state *State, resolved *serverref.Resolved) string {
	if state.Options.Invocation.IsExposedCommand() {
		return fmt.Sprintf("run `%s tools`", state.Options.Invocation.ExposedCommandName)
	}
	if resolved == nil || resolved.Server == nil {
		return ""
	}
	switch {
	case resolved.Server.Source == "ephemeral" && resolved.Server.Command != "":
		return fmt.Sprintf("run `mcp2cli tools --command %q`", resolved.Server.Command)
	case resolved.Server.Source == "ephemeral" && resolved.Server.URL != "":
		return fmt.Sprintf("run `mcp2cli tools --url %q`", resolved.Server.URL)
	case resolved.Server.Name != "":
		return fmt.Sprintf("run `mcp2cli tools %s`", resolved.Server.Name)
	default:
		return ""
	}
}

func toolsInspectHint(state *State, resolved *serverref.Resolved, toolName string) string {
	toolName = naming.ToKebabCase(toolName)
	if state.Options.Invocation.IsExposedCommand() {
		return fmt.Sprintf("run `%s tools %s`", state.Options.Invocation.ExposedCommandName, toolName)
	}
	if resolved == nil || resolved.Server == nil {
		return ""
	}
	switch {
	case resolved.Server.Source == "ephemeral" && resolved.Server.Command != "":
		return fmt.Sprintf("run `mcp2cli tools --command %q %s`", resolved.Server.Command, toolName)
	case resolved.Server.Source == "ephemeral" && resolved.Server.URL != "":
		return fmt.Sprintf("run `mcp2cli tools --url %q %s`", resolved.Server.URL, toolName)
	case resolved.Server.Name != "":
		return fmt.Sprintf("run `mcp2cli tools %s %s`", resolved.Server.Name, toolName)
	default:
		return ""
	}
}

type toolView struct {
	Name        string        `json:"name" yaml:"name"`
	MCPName     string        `json:"mcpName,omitempty" yaml:"mcpName,omitempty"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Args        []toolArgView `json:"args,omitempty" yaml:"args,omitempty"`
	InputSchema any           `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
}

type toolArgView struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	HasDefault  bool   `json:"hasDefault,omitempty" yaml:"hasDefault,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
}

func newToolView(tool types.Tool) toolView {
	cliName := naming.ToKebabCase(tool.Name)
	view := toolView{Name: cliName, Description: tool.Description}
	if cliName != tool.Name {
		view.MCPName = tool.Name
	}
	return view
}
