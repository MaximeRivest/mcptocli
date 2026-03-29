package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/adrg/xdg"
	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/daemon"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/spf13/cobra"
)

func newUpCommand(state *State) *cobra.Command {
	var share bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the server in the background for fast repeated use",
		Long: `Start the server in the background.

By default, the server runs in stdio mode (single client, fastest).
With --share, the server runs in HTTP mode so multiple clients
(terminal, Claude Desktop, notebooks) can share one session.`,
		Example: `  # Start in the background (fast, single client)
  mcp2cli weather up

  # Start in shared HTTP mode (multiple clients)
  mcp2cli weather up --share

  # Then use normally
  mcp2cli weather get-forecast --city Paris`,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := state.BoundServer()
			if err != nil || server == nil {
				return exitcode.New(exitcode.Usage, "up requires a server context (use: mcp2cli <server> up)")
			}
			if server.Command == "" {
				return exitcode.New(exitcode.Config, "up only works with local servers")
			}

			if share {
				return upShared(cmd, server)
			}
			return upStdio(cmd, server)
		},
	}

	cmd.Flags().BoolVar(&share, "share", false, "start in HTTP mode so multiple clients can connect")
	return cmd
}

// upStdio starts the server in stdio mode (current behavior).
func upStdio(cmd *cobra.Command, server *config.Server) error {
	if daemon.IsRunning(xdg.DataHome, server.Name) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s is already running\n", server.Name)
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "find executable")
	}
	child := exec.Command(self, "__daemon", server.Name, server.Command)
	child.Stdout = nil
	child.Stderr = nil
	child.Stdin = nil
	child.SysProcAttr = daemonSysProcAttr()
	if err := child.Start(); err != nil {
		return exitcode.Wrap(exitcode.Transport, err, "start daemon")
	}

	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsRunning(xdg.DataHome, server.Name) {
			fmt.Fprintf(cmd.OutOrStdout(), "%s is running (pid %d)\n", server.Name, child.Process.Pid)
			return nil
		}
	}
	return exitcode.New(exitcode.Transport, "daemon did not start in time")
}

// upShared starts the server in HTTP mode (multiple clients can connect).
func upShared(cmd *cobra.Command, server *config.Server) error {
	if daemon.IsSharedRunning(xdg.DataHome, server.Name) {
		url, _ := daemon.SharedURL(xdg.DataHome, server.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "%s is already running (%s)\n", server.Name, url)
		return nil
	}
	// Also stop any stdio daemon for this server
	if daemon.IsRunning(xdg.DataHome, server.Name) {
		_ = daemon.Stop(xdg.DataHome, server.Name)
	}

	self, err := os.Executable()
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "find executable")
	}
	child := exec.Command(self, "__daemon-shared", server.Name, server.Command)
	child.Stdout = nil
	child.Stderr = nil
	child.Stdin = nil
	child.SysProcAttr = daemonSysProcAttr()
	if err := child.Start(); err != nil {
		return exitcode.Wrap(exitcode.Transport, err, "start shared daemon")
	}

	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsSharedRunning(xdg.DataHome, server.Name) {
			url, _ := daemon.SharedURL(xdg.DataHome, server.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "%s is running (pid %d, %s)\n", server.Name, child.Process.Pid, url)
			return nil
		}
	}
	return exitcode.New(exitcode.Transport, "shared daemon did not start in time")
}

func newDownCommand(state *State) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the background server",
		Example: `  mcp2cli weather down`,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := state.BoundServer()
			if err != nil || server == nil {
				return exitcode.New(exitcode.Usage, "down requires a server context (use: mcp2cli <server> down)")
			}
			// Stop whichever mode is running
			stoppedShared := false
			if daemon.IsSharedRunning(xdg.DataHome, server.Name) {
				if err := daemon.StopShared(xdg.DataHome, server.Name); err == nil {
					stoppedShared = true
				}
			}
			stoppedStdio := false
			if daemon.IsRunning(xdg.DataHome, server.Name) {
				if err := daemon.Stop(xdg.DataHome, server.Name); err == nil {
					stoppedStdio = true
				}
			}
			if !stoppedShared && !stoppedStdio {
				return fmt.Errorf("server %q is not running", server.Name)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s stopped\n", server.Name)
			return nil
		},
	}
}

func newDaemonCommand(state *State) *cobra.Command {
	return &cobra.Command{
		Use:    "__daemon",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			command := args[1]
			return daemon.Run(context.Background(), command, xdg.DataHome, serverName)
		},
	}
}

func newDaemonSharedCommand(state *State) *cobra.Command {
	return &cobra.Command{
		Use:    "__daemon-shared",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			command := args[1]
			return daemon.RunShared(context.Background(), command, xdg.DataHome, serverName)
		},
	}
}
