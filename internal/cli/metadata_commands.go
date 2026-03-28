package cli

import (
	"context"
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
)

func newResourcesCommand(state *State) *cobra.Command {
	var (
		command   string
		urlValue  string
		cwd       string
		envVars   []string
		headers   []string
		authMode  string
		bearerEnv string
		timeout   time.Duration
		output    string
	)
	cmd := &cobra.Command{
		Use:   "resources [server] [resource]",
		Short: "List resources or inspect one resource",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return metadataResourcesCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}
			explicitServer, resourceName, err := parseMetadataArgs(state, args, command, urlValue, "resources [server] [resource]")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: explicitServer, Command: command, URL: urlValue, CWD: cwd, Env: envVars, Headers: headers, Auth: authMode, BearerEnv: bearerEnv}, ctx, cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()
			resources, err := session.ListResources(ctx)
			if err != nil {
				return err
			}
			cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Resources = resources })
			sort.Slice(resources, func(i, j int) bool { return resourceDisplayName(resources[i]) < resourceDisplayName(resources[j]) })
			if resourceName != "" {
				resource, ok := findResource(resources, resourceName)
				if !ok {
					return exitcode.Newf(exitcode.Usage, "resource %q not found", resourceName)
				}
				return renderResourceDescriptor(cmd.OutOrStdout(), resource, outputMode)
			}
			return renderResources(cmd.OutOrStdout(), resources, outputMode)
		},
	}
	addMetadataConnectionFlags(cmd, &command, &urlValue, &cwd, &envVars, &headers, &authMode, &bearerEnv, &timeout, &output)
	return cmd
}

func newResourceCommand(state *State) *cobra.Command {
	var (
		command   string
		urlValue  string
		cwd       string
		envVars   []string
		headers   []string
		authMode  string
		bearerEnv string
		timeout   time.Duration
		output    string
	)
	cmd := &cobra.Command{
		Use:   "resource [server] <resource>",
		Short: "Read a resource",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return metadataResourcesCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}
			explicitServer, resourceName, err := parseMetadataRequiredArgs(state, args, command, urlValue, "resource [server] <resource>")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: explicitServer, Command: command, URL: urlValue, CWD: cwd, Env: envVars, Headers: headers, Auth: authMode, BearerEnv: bearerEnv}, ctx, cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()
			resources, err := session.ListResources(ctx)
			if err != nil {
				return err
			}
			cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Resources = resources })
			resource, ok := findResource(resources, resourceName)
			if !ok {
				return exitcode.Newf(exitcode.Usage, "resource %q not found", resourceName)
			}
			result, err := session.ReadResource(ctx, resource.URI)
			if err != nil {
				return err
			}
			return renderResourceResult(cmd.OutOrStdout(), result, outputMode)
		},
	}
	addMetadataConnectionFlags(cmd, &command, &urlValue, &cwd, &envVars, &headers, &authMode, &bearerEnv, &timeout, &output)
	return cmd
}

func newPromptsCommand(state *State) *cobra.Command {
	var (
		command   string
		urlValue  string
		cwd       string
		envVars   []string
		headers   []string
		authMode  string
		bearerEnv string
		timeout   time.Duration
		output    string
	)
	cmd := &cobra.Command{
		Use:   "prompts [server] [prompt]",
		Short: "List prompts or inspect one prompt",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return metadataPromptsCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}
			explicitServer, promptName, err := parseMetadataArgs(state, args, command, urlValue, "prompts [server] [prompt]")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: explicitServer, Command: command, URL: urlValue, CWD: cwd, Env: envVars, Headers: headers, Auth: authMode, BearerEnv: bearerEnv}, ctx, cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()
			prompts, err := session.ListPrompts(ctx)
			if err != nil {
				return err
			}
			cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Prompts = prompts })
			sort.Slice(prompts, func(i, j int) bool { return promptDisplayName(prompts[i]) < promptDisplayName(prompts[j]) })
			if promptName != "" {
				prompt, ok := findPrompt(prompts, promptName)
				if !ok {
					return exitcode.Newf(exitcode.Usage, "prompt %q not found", promptName)
				}
				return renderPromptDescriptor(cmd.OutOrStdout(), prompt, outputMode)
			}
			return renderPrompts(cmd.OutOrStdout(), prompts, outputMode)
		},
	}
	addMetadataConnectionFlags(cmd, &command, &urlValue, &cwd, &envVars, &headers, &authMode, &bearerEnv, &timeout, &output)
	return cmd
}

func newPromptCommand(state *State) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "prompt [server] <prompt> [args...]",
		Short:              "Render a prompt",
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return promptCommandCompletion(state, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parsePromptInvocationTokens(state, args)
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
			ctx, cancel := context.WithTimeout(cmd.Context(), parsed.Timeout)
			defer cancel()
			resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: parsed.ServerName, Command: parsed.Command, URL: parsed.URL, CWD: parsed.CWD, Env: parsed.Env, Headers: parsed.Headers, Auth: parsed.Auth, BearerEnv: parsed.BearerEnv}, ctx, cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()
			prompts, err := session.ListPrompts(ctx)
			if err != nil {
				return err
			}
			cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) { metadata.Prompts = prompts })
			prompt, ok := findPrompt(prompts, parsed.PromptName)
			if !ok {
				return exitcode.Newf(exitcode.Usage, "prompt %q not found", parsed.PromptName)
			}
			spec := inspect.InspectPrompt(prompt)
			var argMap map[string]any
			if parsed.Input != "" {
				if len(parsed.DynamicArgs) > 0 {
					return exitcode.New(exitcode.Usage, "cannot combine --input with schema-derived prompt arguments")
				}
				argMap, err = loadInputArguments(parsed.Input)
				if err != nil {
					return err
				}
			} else {
				argMap, err = invoke.ParseToolArguments(spec, parsed.DynamicArgs)
				if err != nil {
					return err
				}
			}
			promptArgs := map[string]string{}
			for key, value := range argMap {
				promptArgs[key] = fmt.Sprint(value)
			}
			result, err := session.GetPrompt(ctx, prompt.Name, promptArgs)
			if err != nil {
				return err
			}
			return renderPromptResult(cmd.OutOrStdout(), result, outputMode)
		},
	}
	return cmd
}

type metadataConnectionOptions struct {
	ExplicitName string
	Command      string
	URL          string
	CWD          string
	Env          []string
	Headers      []string
	Auth         string
	BearerEnv    string
}

func openSession(state *State, options metadataConnectionOptions, ctx context.Context, in io.Reader, errOut io.Writer) (*serverref.Resolved, mcpclient.Session, error) {
	repo, err := state.Repo()
	if err != nil {
		return nil, nil, err
	}
	bound, err := state.BoundServer()
	if err != nil {
		return nil, nil, err
	}
	resolved, err := serverref.Resolve(repo, bound, serverref.Options{ExplicitName: options.ExplicitName, Command: options.Command, URL: options.URL, CWD: options.CWD, Env: options.Env, Headers: options.Headers, Auth: options.Auth, BearerEnv: options.BearerEnv})
	if err != nil {
		return nil, nil, err
	}
	store, err := state.TokenStore()
	if err != nil {
		return nil, nil, err
	}
	headers, err := auth.HeadersForServer(store, resolved.Server)
	if err != nil {
		return nil, nil, err
	}
	session, err := mcpclient.Connect(ctx, resolved.Server, headers, terminalConnectOptions(in, errOut))
	if err != nil {
		return nil, nil, err
	}
	return resolved, session, nil
}

func addMetadataConnectionFlags(cmd *cobra.Command, command, urlValue, cwd *string, envVars, headers *[]string, authMode, bearerEnv *string, timeout *time.Duration, output *string) {
	cmd.Flags().StringVar(command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(urlValue, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().StringSliceVar(headers, "header", nil, "Additional HTTP header in Key: Value form (repeatable)")
	cmd.Flags().StringVar(authMode, "auth", "", "Authentication mode override")
	cmd.Flags().StringVar(bearerEnv, "bearer-env", "", "Environment variable containing a bearer token")
	cmd.Flags().DurationVar(timeout, "timeout", mcpclient.DefaultTimeout(), "Request timeout")
	cmd.Flags().StringVarP(output, "output", "o", "auto", "Output format: auto, json, yaml, raw, or table")
}

func parseMetadataArgs(state *State, args []string, command, urlValue, usage string) (string, string, error) {
	if state.Options.Invocation.IsExposedCommand() || command != "" || urlValue != "" {
		switch len(args) {
		case 0:
			return "", "", nil
		case 1:
			return "", args[0], nil
		default:
			return "", "", exitcode.New(exitcode.Usage, "usage: "+usage)
		}
	}
	switch len(args) {
	case 1:
		return args[0], "", nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", exitcode.New(exitcode.Usage, "usage: "+usage)
	}
}

func parseMetadataRequiredArgs(state *State, args []string, command, urlValue, usage string) (string, string, error) {
	explicit, name, err := parseMetadataArgs(state, args, command, urlValue, usage)
	if err != nil {
		return "", "", err
	}
	if name == "" {
		return "", "", exitcode.New(exitcode.Usage, "usage: "+usage)
	}
	return explicit, name, nil
}

func renderResources(w io.Writer, resources []types.Resource, output string) error {
	views := make([]resourceView, 0, len(resources))
	for _, resource := range resources {
		views = append(views, newResourceView(resource))
	}
	switch output {
	case "json":
		return writeJSON(w, views)
	case "yaml":
		return writeYAML(w, views)
	case "raw":
		for _, view := range views {
			fmt.Fprintln(w, view.URI)
		}
		return nil
	case "table", "auto":
		writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, view := range views {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", view.Name, view.URI, strings.TrimSpace(view.Description))
		}
		return writer.Flush()
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func renderResourceDescriptor(w io.Writer, resource types.Resource, output string) error {
	view := newResourceView(resource)
	switch output {
	case "json":
		return writeJSON(w, view)
	case "yaml":
		return writeYAML(w, view)
	case "raw", "table", "auto":
		fmt.Fprintf(w, "NAME\n  %s\n\nURI\n  %s\n", view.Name, view.URI)
		if view.MimeType != "" {
			fmt.Fprintf(w, "\nMIME\n  %s\n", view.MimeType)
		}
		if view.Description != "" {
			fmt.Fprintf(w, "\nDESCRIPTION\n  %s\n", view.Description)
		}
		return nil
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func renderResourceResult(w io.Writer, result *types.ReadResourceResult, output string) error {
	switch output {
	case "json":
		return writeJSON(w, result)
	case "yaml":
		return writeYAML(w, result)
	case "raw", "auto":
		text, ok := textOnlyResource(result)
		if !ok {
			return exitcode.New(exitcode.Usage, "resource is not plain text; use -o json or -o yaml")
		}
		_, err := io.WriteString(w, text)
		if err == nil && !strings.HasSuffix(text, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case "table":
		return writeJSON(w, result)
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func renderPrompts(w io.Writer, prompts []types.Prompt, output string) error {
	views := make([]promptView, 0, len(prompts))
	for _, prompt := range prompts {
		views = append(views, newPromptView(prompt))
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
		writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, view := range views {
			fmt.Fprintf(writer, "%s\t%s\n", view.Name, strings.TrimSpace(view.Description))
		}
		return writer.Flush()
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func renderPromptDescriptor(w io.Writer, prompt types.Prompt, output string) error {
	view := newPromptView(prompt)
	switch output {
	case "json":
		return writeJSON(w, view)
	case "yaml":
		return writeYAML(w, view)
	case "raw", "table", "auto":
		fmt.Fprintf(w, "NAME\n  %s", view.Name)
		if view.Description != "" {
			fmt.Fprintf(w, " - %s", view.Description)
		}
		fmt.Fprintln(w)
		if len(view.Arguments) > 0 {
			fmt.Fprintln(w, "\nARGS")
			writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			for _, arg := range view.Arguments {
				details := arg.Description
				if arg.Required {
					details = "Required. " + details
				}
				fmt.Fprintf(writer, "  --%s string\t%s\n", arg.Name, strings.TrimSpace(details))
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

func renderPromptResult(w io.Writer, result *types.GetPromptResult, output string) error {
	switch output {
	case "json":
		return writeJSON(w, result)
	case "yaml":
		return writeYAML(w, result)
	case "raw", "auto":
		text, ok := textOnlyPrompt(result)
		if !ok {
			return exitcode.New(exitcode.Usage, "prompt result is not plain text; use -o json or -o yaml")
		}
		_, err := io.WriteString(w, text)
		if err == nil && !strings.HasSuffix(text, "\n") {
			_, err = io.WriteString(w, "\n")
		}
		return err
	case "table":
		return writeJSON(w, result)
	default:
		return exitcode.Newf(exitcode.Usage, "unsupported output format %q", output)
	}
}

func parsePromptInvocationTokens(state *State, tokens []string) (*parsedPromptInvocation, error) {
	parsed := &parsedPromptInvocation{Output: "auto", Timeout: mcpclient.DefaultTimeout()}
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
			return nil, exitcode.New(exitcode.Usage, "usage: prompt [server] <prompt> [args...]")
		}
		parsed.PromptName = remaining[0]
		parsed.DynamicArgs = append([]string(nil), remaining[1:]...)
		return parsed, nil
	}
	if len(remaining) < 2 {
		return nil, exitcode.New(exitcode.Usage, "usage: prompt [server] <prompt> [args...]")
	}
	parsed.ServerName = remaining[0]
	parsed.PromptName = remaining[1]
	parsed.DynamicArgs = append([]string(nil), remaining[2:]...)
	return parsed, nil
}

type parsedPromptInvocation struct {
	ServerName  string
	PromptName  string
	Command     string
	URL         string
	CWD         string
	Env         []string
	Headers     []string
	Auth        string
	BearerEnv   string
	Input       string
	Output      string
	Timeout     time.Duration
	DynamicArgs []string
	Help        bool
}

type resourceView struct {
	Name        string `json:"name" yaml:"name"`
	URI         string `json:"uri" yaml:"uri"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty" yaml:"mimeType,omitempty"`
}

type promptView struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Arguments   []promptArgView `json:"arguments,omitempty" yaml:"arguments,omitempty"`
}

type promptArgView struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
}

func newResourceView(resource types.Resource) resourceView {
	name := resourceDisplayName(resource)
	return resourceView{Name: name, URI: resource.URI, Description: resource.Description, MimeType: resource.MimeType}
}

func newPromptView(prompt types.Prompt) promptView {
	view := promptView{Name: promptDisplayName(prompt), Description: prompt.Description}
	for _, arg := range prompt.Arguments {
		view.Arguments = append(view.Arguments, promptArgView{Name: naming.ToKebabCase(arg.Name), Description: arg.Description, Required: arg.Required})
	}
	return view
}

func resourceDisplayName(resource types.Resource) string {
	if strings.TrimSpace(resource.Name) != "" {
		return naming.ToKebabCase(resource.Name)
	}
	if resource.URI == "" {
		return "resource"
	}
	parts := strings.Split(strings.Trim(resource.URI, "/"), "/")
	return naming.ToKebabCase(parts[len(parts)-1])
}

func promptDisplayName(prompt types.Prompt) string { return naming.ToKebabCase(prompt.Name) }

func findResource(resources []types.Resource, requested string) (types.Resource, bool) {
	requested = strings.TrimSpace(requested)
	for _, resource := range resources {
		if resource.URI == requested || resource.Name == requested || naming.ToKebabCase(resource.Name) == naming.ToKebabCase(requested) || resourceDisplayName(resource) == naming.ToKebabCase(requested) {
			return resource, true
		}
	}
	return types.Resource{}, false
}

func findPrompt(prompts []types.Prompt, requested string) (types.Prompt, bool) {
	requestedCLI := naming.ToKebabCase(requested)
	for _, prompt := range prompts {
		if prompt.Name == requested || promptDisplayName(prompt) == requestedCLI {
			return prompt, true
		}
	}
	return types.Prompt{}, false
}

func textOnlyResource(result *types.ReadResourceResult) (string, bool) {
	if result == nil || len(result.Contents) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(result.Contents))
	for _, content := range result.Contents {
		if content.Text == "" {
			return "", false
		}
		parts = append(parts, content.Text)
	}
	return strings.Join(parts, "\n"), true
}

func textOnlyPrompt(result *types.GetPromptResult) (string, bool) {
	if result == nil || len(result.Messages) == 0 {
		return "", false
	}
	parts := make([]string, 0, len(result.Messages))
	for _, message := range result.Messages {
		if message.Content.Type != "text" {
			return "", false
		}
		parts = append(parts, message.Content.Text)
	}
	return strings.Join(parts, "\n\n"), true
}
