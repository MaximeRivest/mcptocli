package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/inconshreveable/mousetrap"
	"github.com/maximerivest/mcptocli/internal/app"
	"github.com/maximerivest/mcptocli/internal/cli"
	"github.com/maximerivest/mcptocli/internal/config"
	"github.com/maximerivest/mcptocli/internal/exitcode"
)

var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func main() {
	code := run()
	if mousetrap.StartedByExplorer() {
		fmt.Println()
		fmt.Println("Press Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(code)
}

func run() int {
	invocation := app.DetectInvocation(os.Args[0])
	args := os.Args[1:]

	// If the first arg is a registered server name (not a known command),
	// treat it as implicit bound-server mode:
	//   mcptocli weather get-forecast 1 2
	// behaves like:
	//   mcp-weather get-forecast 1 2
	if !invocation.IsExposedCommand() && len(args) > 0 {
		first := args[0]
		if first != "" && first[0] != '-' && !app.IsKnownRootCommand(first) {
			if repo, err := config.NewRepository(""); err == nil {
				if _, err := repo.ResolveServer(first); err == nil {
					invocation = app.Invocation{ProgramName: "mcptocli", ExposedCommandName: first, ImplicitBind: true}
					args = args[1:]
				}
			}
		}
	}

	root, err := cli.NewRootCommand(cli.Options{
		Version:    version,
		Commit:     commit,
		BuildDate:  buildDate,
		Invocation: invocation,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, exitcode.Format(err))
		return exitcode.Code(err)
	}

	// When double-clicked from Explorer with no arguments, show help
	// plus a setup hint so the window isn't just a blank flash.
	if mousetrap.StartedByExplorer() && len(args) == 0 {
		root.SetArgs([]string{"--help"})
		_ = root.Execute()
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "To use mcptocli, open a terminal (PowerShell or cmd) and run:")
		fmt.Fprintln(os.Stdout, "  mcptocli add time 'uvx mcp-server-time'")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "To install system-wide, run in PowerShell:")
		fmt.Fprintln(os.Stdout, "  irm https://raw.githubusercontent.com/MaximeRivest/mcptocli/main/install.ps1 | iex")
		return 0
	}

	root.SetArgs(app.RewriteArgsForExposedMode(invocation, args))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, exitcode.Format(err))
		return exitcode.Code(err)
	}

	return 0
}
