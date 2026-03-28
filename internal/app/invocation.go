package app

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Invocation describes how the program was invoked.
type Invocation struct {
	ProgramName        string
	ExposedCommandName string
	ImplicitBind       bool // true when server name used as first arg, not via exposed shim
}

// DetectInvocation determines whether the binary was invoked as mcp2cli or as
// an exposed command like mcp-weather or wea.
func DetectInvocation(argv0 string) Invocation {
	base := filepath.Base(argv0)
	base = stripExecutableSuffix(base)

	invocation := Invocation{ProgramName: base}
	if base == "" || base == "mcp2cli" {
		return invocation
	}

	invocation.ExposedCommandName = base
	return invocation
}

// IsExposedCommand reports whether the current invocation is through an exposed command.
func (i Invocation) IsExposedCommand() bool {
	return i.ExposedCommandName != ""
}

// RewriteArgsForExposedMode rewrites shorthand tool calls in exposed-command
// mode. For example:
//
//	wea get-forecast --latitude 1 --longitude 2
//
// becomes:
//
//	wea tool get-forecast --latitude 1 --longitude 2
func RewriteArgsForExposedMode(inv Invocation, args []string) []string {
	if !inv.IsExposedCommand() || len(args) == 0 {
		return args
	}

	first := args[0]
	if first == "" || strings.HasPrefix(first, "-") || IsReservedExposedCommand(first) {
		return args
	}

	rewritten := make([]string, 0, len(args)+1)
	rewritten = append(rewritten, "tool")
	rewritten = append(rewritten, args...)
	return rewritten
}

// IsReservedExposedCommand reports whether an exposed-command argument should be
// interpreted as a meta-command rather than a tool shorthand.
func IsReservedExposedCommand(name string) bool {
	switch name {
	case "help", "completion", "version", "login", "tools", "tool", "resources", "resource", "prompts", "prompt", "shell", "doctor":
		return true
	default:
		return false
	}
}

// IsKnownRootCommand reports whether a name is a built-in mcp2cli subcommand.
func IsKnownRootCommand(name string) bool {
	switch name {
	case "help", "completion", "version", "add", "ls", "rm", "expose", "unexpose",
		"login", "tools", "tool", "resources", "resource", "prompts", "prompt",
		"shell", "doctor":
		return true
	default:
		return false
	}
}

func stripExecutableSuffix(name string) string {
	if runtime.GOOS != "windows" {
		return name
	}

	suffixes := []string{".exe", ".cmd", ".bat", ".ps1"}
	lower := strings.ToLower(name)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return name[:len(name)-len(suffix)]
		}
	}

	return name
}
