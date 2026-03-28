package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	mcpclient "github.com/maximerivest/mcp2cli/internal/mcp/client"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
	"github.com/maximerivest/mcp2cli/internal/serverref"
	"github.com/spf13/cobra"
)

func newToolsCommand(state *State) *cobra.Command {
	var (
		command string
		url     string
		cwd     string
		envVars []string
		timeout time.Duration
		output  string
	)

	cmd := &cobra.Command{
		Use:   "tools [server] [tool]",
		Short: "List tools or inspect one tool",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			resolved, err := serverref.Resolve(repo, bound, explicitServer, command, url, cwd, envVars)
			if err != nil {
				return err
			}
			if resolved.Server.URL != "" {
				return fmt.Errorf("HTTP MCP servers are not implemented yet")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			client, err := mcpclient.ConnectStdio(ctx, resolved.Server)
			if err != nil {
				return err
			}
			defer func() { _ = client.Close() }()

			tools, err := client.ListTools(ctx)
			if err != nil {
				return err
			}
			sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

			if toolName != "" {
				for _, tool := range tools {
					if tool.Name == toolName {
						return renderTool(cmd, tool, output)
					}
				}
				return fmt.Errorf("tool %q not found", toolName)
			}
			return renderTools(cmd, tools, output)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(&url, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().DurationVar(&timeout, "timeout", mcpclient.DefaultTimeout(), "Request timeout")
	cmd.Flags().StringVarP(&output, "output", "o", "auto", "Output format: auto or json")
	return cmd
}

func newToolCommand(state *State) *cobra.Command {
	var (
		command string
		url     string
		cwd     string
		envVars []string
		input   string
		timeout time.Duration
		output  string
	)

	cmd := &cobra.Command{
		Use:   "tool [server] <tool> [args...]",
		Short: "Invoke a tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			explicitServer, toolName, err := parseToolArgs(state, args, command, url)
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
			resolved, err := serverref.Resolve(repo, bound, explicitServer, command, url, cwd, envVars)
			if err != nil {
				return err
			}
			if resolved.Server.URL != "" {
				return fmt.Errorf("HTTP MCP servers are not implemented yet")
			}

			arguments, err := loadInputArguments(input)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			client, err := mcpclient.ConnectStdio(ctx, resolved.Server)
			if err != nil {
				return err
			}
			defer func() { _ = client.Close() }()

			result, err := client.CallTool(ctx, toolName, arguments)
			if err != nil {
				return err
			}
			return renderToolResult(cmd, result, output)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Local server command to run")
	cmd.Flags().StringVar(&url, "url", "", "Remote MCP server URL")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for local server commands")
	cmd.Flags().StringSliceVar(&envVars, "env", nil, "Environment variable override in KEY=VALUE form (repeatable)")
	cmd.Flags().StringVar(&input, "input", "{}", "JSON object, @file.json, or @- for stdin")
	cmd.Flags().DurationVar(&timeout, "timeout", mcpclient.DefaultTimeout(), "Request timeout")
	cmd.Flags().StringVarP(&output, "output", "o", "auto", "Output format: auto or json")
	return cmd
}

func parseToolsArgs(state *State, args []string, command string, url string) (string, string, error) {
	if state.Options.Invocation.IsExposedCommand() || command != "" || url != "" {
		switch len(args) {
		case 0:
			return "", "", nil
		case 1:
			return "", args[0], nil
		default:
			return "", "", fmt.Errorf("usage: tools [server] [tool]")
		}
	}

	switch len(args) {
	case 1:
		return args[0], "", nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("usage: tools [server] [tool]")
	}
}

func parseToolArgs(state *State, args []string, command string, url string) (string, string, error) {
	if state.Options.Invocation.IsExposedCommand() || command != "" || url != "" {
		if len(args) != 1 {
			return "", "", fmt.Errorf("usage: tool [server] <tool> [args...]")
		}
		return "", args[0], nil
	}
	if len(args) != 2 {
		return "", "", fmt.Errorf("usage: tool [server] <tool> [args...]")
	}
	return args[0], args[1], nil
}

func renderTools(cmd *cobra.Command, tools []types.Tool, output string) error {
	if output == "json" {
		return writeJSON(cmd.OutOrStdout(), tools)
	}

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	for _, tool := range tools {
		description := strings.TrimSpace(tool.Description)
		fmt.Fprintf(writer, "%s\t%s\n", tool.Name, description)
	}
	return writer.Flush()
}

func renderTool(cmd *cobra.Command, tool types.Tool, output string) error {
	if output == "json" {
		return writeJSON(cmd.OutOrStdout(), tool)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "NAME\n  %s", tool.Name)
	if tool.Description != "" {
		fmt.Fprintf(cmd.OutOrStdout(), " - %s", tool.Description)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	if len(tool.InputSchema) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nINPUT SCHEMA")
		var pretty any
		if err := json.Unmarshal(tool.InputSchema, &pretty); err == nil {
			payload, _ := json.MarshalIndent(pretty, "  ", "  ")
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", strings.ReplaceAll(string(payload), "\n", "\n  "))
		}
	}
	return nil
}

func renderToolResult(cmd *cobra.Command, result *types.CallToolResult, output string) error {
	if output == "json" {
		return writeJSON(cmd.OutOrStdout(), result)
	}
	if len(result.Content) > 0 {
		allText := true
		texts := make([]string, 0, len(result.Content))
		for _, item := range result.Content {
			if item.Type != "text" {
				allText = false
				break
			}
			texts = append(texts, item.Text)
		}
		if allText {
			for i, text := range texts {
				if i > 0 {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				fmt.Fprint(cmd.OutOrStdout(), text)
			}
			if len(texts) > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}
	}
	return writeJSON(cmd.OutOrStdout(), result)
}

func loadInputArguments(input string) (map[string]any, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return map[string]any{}, nil
	}

	var data []byte
	var err error
	if strings.HasPrefix(input, "@") {
		name := strings.TrimPrefix(input, "@")
		switch name {
		case "-":
			data, err = ioReadAll(os.Stdin)
		default:
			data, err = os.ReadFile(name)
		}
		if err != nil {
			return nil, fmt.Errorf("read input %q: %w", input, err)
		}
	} else {
		data = []byte(input)
	}

	arguments := map[string]any{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return arguments, nil
	}
	if err := json.Unmarshal(data, &arguments); err != nil {
		return nil, fmt.Errorf("parse --input as JSON object: %w", err)
	}
	return arguments, nil
}

func writeJSON(w io.Writer, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(payload, '\n'))
	return err
}

func ioReadAll(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}
