package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNotImplementedCommand(state *State, use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.Options.Invocation.IsExposedCommand() {
				server, err := state.BoundServer()
				if err != nil {
					return err
				}
				return fmt.Errorf("%s for server %q is not implemented yet", cmd.Name(), server.Name)
			}
			return fmt.Errorf("%s is not implemented yet", cmd.Name())
		},
	}
}
