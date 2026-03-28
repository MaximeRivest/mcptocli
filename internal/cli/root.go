package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the CLI root.
func NewRootCommand(opts Options) (*cobra.Command, error) {
	state, err := newState(opts)
	if err != nil {
		return nil, err
	}

	use := "mcp2cli"
	short := "Turn any MCP server into a CLI"
	long := "mcp2cli turns MCP servers into delightful command-line tools."
	if opts.Invocation.IsExposedCommand() {
		use = opts.Invocation.ExposedCommandName
		short = fmt.Sprintf("Bound CLI for exposed command %q", opts.Invocation.ExposedCommandName)
		long = fmt.Sprintf("%s is an exposed mcp2cli command bound to a registered MCP server.", opts.Invocation.ExposedCommandName)
	}

	root := &cobra.Command{
		Use:           use,
		Short:         short,
		Long:          long,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !state.Options.Invocation.IsExposedCommand() {
			return nil
		}
		if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version" {
			return nil
		}
		_, err := state.BoundServer()
		return err
	}

	root.AddCommand(newVersionCommand(state))
	root.AddCommand(newCompletionCommand(root))

	if !opts.Invocation.IsExposedCommand() {
		root.AddCommand(newAddCommand(state))
		root.AddCommand(newListCommand(state))
		root.AddCommand(newRemoveCommand(state))
		root.AddCommand(newExposeCommand(state))
		root.AddCommand(newUnexposeCommand(state))
	}

	root.AddCommand(newLoginCommand(state))
	root.AddCommand(newToolsCommand(state))
	root.AddCommand(newToolCommand(state))
	root.AddCommand(newResourcesCommand(state))
	root.AddCommand(newResourceCommand(state))
	root.AddCommand(newPromptsCommand(state))
	root.AddCommand(newPromptCommand(state))
	root.AddCommand(newNotImplementedCommand(state, "shell", "Open an interactive MCP shell"))
	root.AddCommand(newDoctorCommand(state))

	return root, nil
}
