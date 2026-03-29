package cli

import (
	"os"

	"golang.org/x/term"
)

// ANSI escape codes for styling.
const (
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiItalic  = "\033[3m"
	ansiReset   = "\033[0m"
	ansiCyan    = "\033[36m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiWhite   = "\033[97m"
)

// colorEnabled returns true if stdout supports ANSI colors.
func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// style helpers — return input unchanged if color is disabled.

func bold(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

func dim(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiDim + s + ansiReset
}

func cyan(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiCyan + s + ansiReset
}

func green(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiGreen + s + ansiReset
}

func yellow(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiYellow + s + ansiReset
}

func boldCyan(s string) string {
	if !colorEnabled() {
		return s
	}
	return ansiBold + ansiCyan + s + ansiReset
}
