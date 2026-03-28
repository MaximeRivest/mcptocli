package main

import (
	"fmt"
	"os"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/cli"
	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
)

var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func main() {
	os.Exit(run())
}

func run() int {
	invocation := app.DetectInvocation(os.Args[0])
	args := os.Args[1:]

	// If the first arg is a registered server name (not a known command),
	// treat it as implicit bound-server mode:
	//   mcp2cli weather get-forecast 1 2
	// behaves like:
	//   mcp-weather get-forecast 1 2
	if !invocation.IsExposedCommand() && len(args) > 0 {
		first := args[0]
		if first != "" && first[0] != '-' && !app.IsKnownRootCommand(first) {
			if repo, err := config.NewRepository(""); err == nil {
				if _, err := repo.ResolveServer(first); err == nil {
					invocation = app.Invocation{ProgramName: "mcp2cli", ExposedCommandName: first, ImplicitBind: true}
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

	root.SetArgs(app.RewriteArgsForExposedMode(invocation, args))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, exitcode.Format(err))
		return exitcode.Code(err)
	}

	return 0
}
