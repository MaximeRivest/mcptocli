package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	groupServers = "servers"
	groupUse     = "use"
	groupDaemon  = "daemon"
)

// NewRootCommand builds the CLI root.
func NewRootCommand(opts Options) (*cobra.Command, error) {
	state, err := newState(opts)
	if err != nil {
		return nil, err
	}

	use := "mcp2cli"
	short := "Turn any MCP server into a CLI"
	long := `mcp2cli turns MCP servers into delightful command-line tools.

Quick start:
  mcp2cli add time 'npx -y @modelcontextprotocol/server-time'
  mcp2cli time tools
  mcp2cli time get-current-time --timezone America/New_York`
	if opts.Invocation.IsExposedCommand() {
		use = opts.Invocation.ExposedCommandName
		short = fmt.Sprintf("Bound CLI for exposed command %q", opts.Invocation.ExposedCommandName)
		long = fmt.Sprintf(`%s is an exposed mcp2cli command bound to a registered MCP server.

Quick start:
  %s tools
  %s <tool> [args...]
  %s shell`, opts.Invocation.ExposedCommandName, use, use, use)
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

	// Define command groups so the help page is organized by workflow.
	if !opts.Invocation.IsExposedCommand() {
		root.AddGroup(
			&cobra.Group{ID: groupServers, Title: "Server Management:"},
			&cobra.Group{ID: groupUse, Title: "Use a Server:"},
			&cobra.Group{ID: groupDaemon, Title: "Background / Sharing:"},
		)
	}

	root.AddCommand(newVersionCommand(state))
	root.AddCommand(newCompletionCommand(root))

	if !opts.Invocation.IsExposedCommand() {
		addCmd := newAddCommand(state)
		addCmd.GroupID = groupServers
		lsCmd := newListCommand(state)
		lsCmd.GroupID = groupServers
		rmCmd := newRemoveCommand(state)
		rmCmd.GroupID = groupServers
		exposeCmd := newExposeCommand(state)
		exposeCmd.GroupID = groupServers

		root.AddCommand(addCmd, lsCmd, rmCmd, exposeCmd)
	}

	toolsCmd := newToolsCommand(state)
	toolCmd := newToolCommand(state)
	resourcesCmd := newResourcesCommand(state)
	resourceCmd := newResourceCommand(state)
	promptsCmd := newPromptsCommand(state)
	promptCmd := newPromptCommand(state)
	shellCmd := newShellCommand(state)
	loginCmd := newLoginCommand(state)
	doctorCmd := newDoctorCommand(state)
	upCmd := newUpCommand(state)
	downCmd := newDownCommand(state)

	if !opts.Invocation.IsExposedCommand() {
		toolsCmd.GroupID = groupUse
		toolCmd.GroupID = groupUse
		resourcesCmd.GroupID = groupUse
		resourceCmd.GroupID = groupUse
		promptsCmd.GroupID = groupUse
		promptCmd.GroupID = groupUse
		shellCmd.GroupID = groupUse
		loginCmd.GroupID = groupUse
		doctorCmd.GroupID = groupUse
		upCmd.GroupID = groupDaemon
		downCmd.GroupID = groupDaemon
	}

	root.AddCommand(toolsCmd, toolCmd, resourcesCmd, resourceCmd,
		promptsCmd, promptCmd, shellCmd, loginCmd, doctorCmd,
		upCmd, downCmd)
	root.AddCommand(newDaemonCommand(state))
	root.AddCommand(newDaemonSharedCommand(state))

	return root, nil
}
