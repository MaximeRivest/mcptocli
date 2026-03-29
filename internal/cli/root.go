package cli

import (
	"fmt"

	"github.com/maximerivest/mcptocli/internal/cache"
	"github.com/maximerivest/mcptocli/internal/naming"
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

	use := opts.Invocation.ProgramName
	if use == "" {
		use = "mcptocli"
	}
	short := "Turn any MCP server into a CLI"
	long := fmt.Sprintf(`%s turns MCP servers into delightful command-line tools.

Quick start:
  %s add time 'uvx mcp-server-time'
  %s time tools
  %s time get-current-time --timezone America/New_York`, use, use, use, use)
	if opts.Invocation.IsExposedCommand() {
		use = opts.Invocation.ExposedCommandName
		long = exposedHelpText(state, use)
		short = use
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

	if opts.Invocation.IsExposedCommand() {
		root.AddGroup(
			&cobra.Group{ID: groupUse, Title: "Tools:"},
			&cobra.Group{ID: groupDaemon, Title: "Commands:"},
		)
	} else {
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

	if opts.Invocation.IsExposedCommand() {
		// For exposed commands, add cached tools as top-level subcommands
		addCachedToolCommands(state, root)

		// Useful commands visible in help
		shellCmd.GroupID = groupDaemon
		upCmd.GroupID = groupDaemon
		downCmd.GroupID = groupDaemon
		toolsCmd.GroupID = groupDaemon
		loginCmd.GroupID = groupDaemon
		doctorCmd.GroupID = groupDaemon

		// Plumbing commands — still available but hidden from help
		toolCmd.Hidden = true
		resourcesCmd.Hidden = true
		resourceCmd.Hidden = true
		promptsCmd.Hidden = true
		promptCmd.Hidden = true
	} else {
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

// exposedHelpText builds a short help string for exposed commands.
func exposedHelpText(state *State, use string) string {
	metadata := loadExposedMetadata(state)
	if metadata == nil || len(metadata.Tools) == 0 {
		return fmt.Sprintf("Run \"%s tools\" to discover available tools.", use)
	}

	first := naming.ToKebabCase(metadata.Tools[0].Name)
	return fmt.Sprintf("Examples:\n  %s %s --help\n  %s shell\n  %s up", use, first, use, use)
}

// loadExposedMetadata loads cached metadata for the bound server.
func loadExposedMetadata(state *State) *cache.Metadata {
	server, err := state.BoundServer()
	if err != nil || server == nil {
		return nil
	}
	store, err := state.MetadataStore()
	if err != nil || store == nil {
		return nil
	}
	metadata, _ := store.Load(server)
	return metadata
}

// addCachedToolCommands adds cached tools as top-level subcommands for exposed commands.
// Each tool appears in the help listing. Actual invocation and --help display are
// handled by the "tool" command (which renders schema-based help for exposed commands).
func addCachedToolCommands(state *State, root *cobra.Command) {
	metadata := loadExposedMetadata(state)
	if metadata == nil {
		return
	}
	for _, t := range metadata.Tools {
		toolName := naming.ToKebabCase(t.Name)
		originalName := t.Name
		desc := t.Description
		if desc == "" {
			desc = "(no description)"
		}

		capturedOriginalName := originalName

		stubCmd := &cobra.Command{
			Use:                toolName + " [args...]",
			Short:              desc,
			GroupID:            groupUse,
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				toolCmd := newToolCommand(state)
				newArgs := append([]string{capturedOriginalName}, args...)
				toolCmd.SetArgs(newArgs)
				return toolCmd.Execute()
			},
		}
		root.AddCommand(stubCmd)
	}
}
