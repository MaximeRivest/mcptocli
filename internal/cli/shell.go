package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chzyer/readline"
	"github.com/kballard/go-shellquote"
	"github.com/maximerivest/mcptocli/internal/cache"
	"github.com/maximerivest/mcptocli/internal/config"
	"github.com/maximerivest/mcptocli/internal/exitcode"
	"github.com/maximerivest/mcptocli/internal/invoke"
	mcpclient "github.com/maximerivest/mcptocli/internal/mcp/client"
	"github.com/maximerivest/mcptocli/internal/mcp/types"
	"github.com/maximerivest/mcptocli/internal/naming"
	"github.com/maximerivest/mcptocli/internal/schema/inspect"
	"github.com/maximerivest/mcptocli/internal/serverref"
	"github.com/spf13/cobra"
)

var errShellExit = errors.New("shell exit")

type shellEnv struct {
	state     *State
	resolved  *serverref.Resolved
	session   mcpclient.Session
	tools     []types.Tool
	resources []types.Resource
	prompts   []types.Prompt
	output    string
	writer    io.Writer
}

func newShellCommand(state *State) *cobra.Command {
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
		Use:   "shell [server]",
		Short: "Open an interactive MCP shell",
		Long: `Open an interactive shell connected to an MCP server.

The shell supports tab completion, history, and all tool/resource/prompt
commands. Type "help" inside the shell for available commands.`,
		Example: `  # Open a shell for a registered server
  mcptocli shell weather

  # Open a shell for a one-off server
  mcptocli shell --url https://mcp.example.com/sse`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputMode, err := normalizeOutputMode(output)
			if err != nil {
				return err
			}
			explicitServer := ""
			if !state.Options.Invocation.IsExposedCommand() && command == "" && urlValue == "" {
				if len(args) != 1 {
					return exitcode.New(exitcode.Usage, "usage: shell [server]")
				}
				explicitServer = args[0]
			} else if len(args) > 0 {
				return exitcode.New(exitcode.Usage, "usage: shell [server]")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: explicitServer, Command: command, URL: urlValue, CWD: cwd, Env: envVars, Headers: headers, Auth: authMode, BearerEnv: bearerEnv}, ctx, os.Stdin, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			defer func() { _ = session.Close() }()

			env, err := newShellEnv(state, resolved, session, outputMode, cmd.OutOrStdout(), ctx)
			if err != nil {
				return err
			}
			return runInteractiveShell(env)
		},
	}
	addMetadataConnectionFlags(cmd, &command, &urlValue, &cwd, &envVars, &headers, &authMode, &bearerEnv, &timeout, &output)
	return cmd
}

func newShellEnv(state *State, resolved *serverref.Resolved, session mcpclient.Session, output string, writer io.Writer, ctx context.Context) (*shellEnv, error) {
	tools, err := session.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := session.ListResources(ctx)
	if err != nil {
		return nil, err
	}
	prompts, err := session.ListPrompts(ctx)
	if err != nil {
		return nil, err
	}
	cacheMetadata(state, resolved.Server, func(metadata *cache.Metadata) {
		metadata.Tools = tools
		metadata.Resources = resources
		metadata.Prompts = prompts
	})
	return &shellEnv{state: state, resolved: resolved, session: session, tools: tools, resources: resources, prompts: prompts, output: output, writer: writer}, nil
}

func runInteractiveShell(env *shellEnv) error {
	repo, err := env.state.Repo()
	if err != nil {
		return err
	}
	historyFile := shellHistoryPath(repo.Paths, env.resolved.Server)
	if err := os.MkdirAll(filepath.Dir(historyFile), 0o755); err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "create shell history directory")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          shellPrompt(env.resolved.Server) + "> ",
		HistoryFile:     historyFile,
		AutoComplete:    buildShellCompleter(env),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "start shell")
	}
	defer rl.Close()

	// Welcome banner
	printShellBanner(rl.Stdout(), env)

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if strings.TrimSpace(line) == "" {
				continue
			}
			err = nil
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return exitcode.Wrap(exitcode.Internal, err, "read shell input")
		}
		if err := dispatchShellLine(env, line); err != nil {
			if errors.Is(err, errShellExit) {
				return nil
			}
			fmt.Fprintln(env.writer, exitcode.Format(err))
		}
	}
}

func printShellBanner(w io.Writer, env *shellEnv) {
	name := shellPrompt(env.resolved.Server)
	fmt.Fprintf(w, "Connected to %s\n", bold(name))

	if len(env.tools) > 0 {
		fmt.Fprintf(w, "\n%s (%d):\n", bold("Tools"), len(env.tools))
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, t := range env.tools {
			cliName := naming.ToKebabCase(t.Name)
			desc := truncateDescription(t.Description)
			fmt.Fprintf(tw, "  %s\t%s\n", cyan(cliName), dim(desc))
		}
		tw.Flush()
	}

	if len(env.resources) > 0 {
		fmt.Fprintf(w, "\n%s (%d):\n", bold("Resources"), len(env.resources))
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, r := range env.resources {
			desc := truncateDescription(r.Name)
			if r.Description != "" {
				desc = truncateDescription(r.Description)
			}
			fmt.Fprintf(tw, "  %s\t%s\n", cyan(r.URI), dim(desc))
		}
		tw.Flush()
	}

	if len(env.prompts) > 0 {
		fmt.Fprintf(w, "\n%s (%d):\n", bold("Prompts"), len(env.prompts))
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, p := range env.prompts {
			desc := truncateDescription(p.Description)
			fmt.Fprintf(tw, "  %s\t%s\n", cyan(p.Name), dim(desc))
		}
		tw.Flush()
	}

	fmt.Fprintf(w, "\nType %s for commands, %s to quit.\n\n", bold("help"), bold("exit"))
}

func dispatchShellLine(env *shellEnv, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	args, err := shellquote.Split(line)
	if err != nil {
		return exitcode.Wrap(exitcode.Usage, err, "parse shell input")
	}
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "exit", "quit":
		return errShellExit
	case "help":
		return renderShellHelp(env.writer)
	case "tools":
		return renderTools(env.writer, env.tools, env.output)
	case "resources":
		return renderResources(env.writer, env.resources, env.output)
	case "prompts":
		return renderPrompts(env.writer, env.prompts, env.output)
	case "set":
		if len(args) == 3 && args[1] == "output" {
			outputMode, err := normalizeOutputMode(args[2])
			if err != nil {
				return err
			}
			env.output = outputMode
			_, err = fmt.Fprintf(env.writer, "output = %s\n", outputMode)
			return err
		}
		return exitcode.New(exitcode.Usage, "usage: set output <auto|json|yaml|raw|table>")
	case "tool":
		if len(args) < 2 {
			return exitcode.New(exitcode.Usage, "usage: tool <name> [args...]")
		}
		return shellInvokeTool(env, args[1], args[2:])
	case "resource":
		if len(args) != 2 {
			return exitcode.New(exitcode.Usage, "usage: resource <name>")
		}
		return shellReadResource(env, args[1])
	case "prompt":
		if len(args) < 2 {
			return exitcode.New(exitcode.Usage, "usage: prompt <name> [args...]")
		}
		return shellRenderPrompt(env, args[1], args[2:])
	default:
		return shellInvokeTool(env, args[0], args[1:])
	}
}

func shellInvokeTool(env *shellEnv, name string, tokens []string) error {
	tool, ok := findTool(env.tools, name)
	if !ok {
		return exitcode.Newf(exitcode.Usage, "tool %q not found", name)
	}
	spec, err := inspect.InspectTool(tool)
	if err != nil {
		return err
	}
	arguments, err := invoke.ParseToolArguments(spec, tokens)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := env.session.CallTool(ctx, tool.Name, arguments)
	if err != nil {
		return err
	}
	return renderToolResult(env.writer, result, env.output)
}

func shellReadResource(env *shellEnv, name string) error {
	resource, ok := findResource(env.resources, name)
	if !ok {
		return exitcode.Newf(exitcode.Usage, "resource %q not found", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := env.session.ReadResource(ctx, resource.URI)
	if err != nil {
		return err
	}
	return renderResourceResult(env.writer, result, env.output)
}

func shellRenderPrompt(env *shellEnv, name string, tokens []string) error {
	prompt, ok := findPrompt(env.prompts, name)
	if !ok {
		return exitcode.Newf(exitcode.Usage, "prompt %q not found", name)
	}
	spec := inspect.InspectPrompt(prompt)
	arguments, err := invoke.ParseToolArguments(spec, tokens)
	if err != nil {
		return err
	}
	promptArgs := map[string]string{}
	for key, value := range arguments {
		promptArgs[key] = fmt.Sprint(value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := env.session.GetPrompt(ctx, prompt.Name, promptArgs)
	if err != nil {
		return err
	}
	return renderPromptResult(env.writer, result, env.output)
}

func buildShellCompleter(env *shellEnv) *readline.PrefixCompleter {
	items := []readline.PrefixCompleterInterface{
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("tools"),
		readline.PcItem("resources"),
		readline.PcItem("prompts"),
		readline.PcItem("set", readline.PcItem("output", readline.PcItem("auto"), readline.PcItem("json"), readline.PcItem("yaml"), readline.PcItem("raw"), readline.PcItem("table"))),
	}

	resourceItems := []readline.PrefixCompleterInterface{}
	for _, resource := range env.resources {
		resourceItems = append(resourceItems, readline.PcItem(resourceDisplayName(resource)))
	}
	items = append(items, readline.PcItem("resource", resourceItems...))

	promptItems := []readline.PrefixCompleterInterface{}
	for _, prompt := range env.prompts {
		children := []readline.PrefixCompleterInterface{}
		for _, arg := range inspect.InspectPrompt(prompt).Arguments {
			children = append(children, readline.PcItem("--"+arg.CLIName))
		}
		promptItems = append(promptItems, readline.PcItem(promptDisplayName(prompt), children...))
	}
	items = append(items, readline.PcItem("prompt", promptItems...))

	toolItems := []readline.PrefixCompleterInterface{}
	for _, tool := range env.tools {
		spec, err := inspect.InspectTool(tool)
		if err != nil {
			continue
		}
		children := []readline.PrefixCompleterInterface{}
		for _, arg := range spec.Arguments {
			children = append(children, readline.PcItem("--"+arg.CLIName))
			if arg.Type == "boolean" {
				children = append(children, readline.PcItem("--no-"+arg.CLIName))
			}
		}
		name := naming.ToKebabCase(tool.Name)
		toolItems = append(toolItems, readline.PcItem(name, children...))
		items = append(items, readline.PcItem(name, children...))
	}
	items = append(items, readline.PcItem("tool", toolItems...))
	return readline.NewPrefixCompleter(items...)
}

func renderShellHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, "Commands: tools, resources, resource <name>, prompts, prompt <name> [args...], tool <name> [args...], <tool> [args...], set output <mode>, help, exit")
	return err
}

func shellPrompt(server *config.Server) string {
	if server == nil {
		return "mcp"
	}
	if server.Name != "" && server.Name != "(direct)" {
		return server.Name
	}
	if server.URL != "" {
		return "remote"
	}
	return "server"
}

func shellHistoryPath(paths config.Paths, server *config.Server) string {
	base := filepath.Dir(paths.ExposeBinDir)
	name := shellPrompt(server)
	return filepath.Join(base, "history", name+".txt")
}
