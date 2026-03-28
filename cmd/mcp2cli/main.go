package main

import (
	"fmt"
	"os"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/cli"
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

	root.SetArgs(app.RewriteArgsForExposedMode(invocation, os.Args[1:]))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, exitcode.Format(err))
		return exitcode.Code(err)
	}

	return 0
}
