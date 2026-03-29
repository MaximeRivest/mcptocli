package cli

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const connectionFlagGroup = "connection"

// markConnectionFlag annotates a flag so the custom help template can group it.
func markConnectionFlag(cmd *cobra.Command, name string) {
	f := cmd.Flags().Lookup(name)
	if f == nil {
		return
	}
	if f.Annotations == nil {
		f.Annotations = map[string][]string{}
	}
	f.Annotations["group"] = []string{connectionFlagGroup}
}

// useGroupedHelp installs a help template that separates connection flags
// from the primary flags on commands that have both.
func useGroupedHelp(cmd *cobra.Command) {
	cmd.SetHelpFunc(groupedHelpFunc)
}

func groupedHelpFunc(cmd *cobra.Command, args []string) {
	t := template.New("help")
	t.Funcs(template.FuncMap{
		"trimTrailingWhitespaces": func(s string) string { return strings.TrimRightFunc(s, func(r rune) bool { return r == ' ' || r == '\n' }) },
		"rpad":                    rpad,
		"primaryFlags":            primaryFlagUsages,
		"connectionFlags":         connectionFlagUsages,
		"hasConnectionFlags":      hasConnectionFlags,
	})
	template.Must(t.Parse(groupedHelpTemplate))

	var buf bytes.Buffer
	if err := t.Execute(&buf, cmd); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return
	}
	fmt.Fprint(cmd.OutOrStdout(), buf.String())
}

func rpad(s string, padding int) string {
	f := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(f, s)
}

func primaryFlagUsages(cmd *cobra.Command) string {
	return flagUsagesFiltered(cmd.LocalFlags(), false)
}

func connectionFlagUsages(cmd *cobra.Command) string {
	return flagUsagesFiltered(cmd.LocalFlags(), true)
}

func hasConnectionFlags(cmd *cobra.Command) bool {
	has := false
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if isConnectionFlag(f) {
			has = true
		}
	})
	return has
}

func isConnectionFlag(f *pflag.Flag) bool {
	if f.Annotations == nil {
		return false
	}
	groups, ok := f.Annotations["group"]
	if !ok {
		return false
	}
	for _, g := range groups {
		if g == connectionFlagGroup {
			return true
		}
	}
	return false
}

func flagUsagesFiltered(flags *pflag.FlagSet, wantConnection bool) string {
	filtered := pflag.NewFlagSet("filtered", pflag.ContinueOnError)
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if isConnectionFlag(f) == wantConnection {
			filtered.AddFlag(f)
		}
	})
	return filtered.FlagUsages()
}

const groupedHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .Runnable}}Usage:
  {{.UseLine}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{ primaryFlags . | trimTrailingWhitespaces}}{{if hasConnectionFlags .}}

Connection Flags:
{{ connectionFlags . | trimTrailingWhitespaces}}{{end}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
