package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestToolsCommandWithRegisteredStdioServer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tools", "weather"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr: %s", err, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "echo") || !strings.Contains(output, "get-forecast") {
		t.Fatalf("unexpected tools output: %s", output)
	}
}

func TestToolCommandWithInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "weather", "echo", "--input", `{"message":"hello"}`})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "echo: hello" {
		t.Fatalf("tool output = %q, want %q", got, "echo: hello")
	}
}

func TestToolCommandWithElicitation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetIn(strings.NewReader("y\n"))
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "weather", "confirm-action", "--action", "deploy"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute elicitation: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "Action 'deploy' confirmed and executed!" {
		t.Fatalf("elicitation output = %q", got)
	}
}

func TestToolCommandWithElicitationRequiresInteractiveInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	file, err := os.CreateTemp(t.TempDir(), "noninteractive")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer file.Close()
	root.SetIn(file)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "weather", "confirm-action", "--action", "deploy"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected elicitation error")
	}
}

func TestToolCommandWithSchemaFlagsAndPositionals(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "weather", "get-forecast", "--latitude", "37.7", "--longitude", "-122.4", "-o", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute flags: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"latitude": 37.7`) || !strings.Contains(stdout.String(), `"longitude": -122.4`) {
		t.Fatalf("unexpected JSON output: %s", stdout.String())
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "weather", "get-forecast", "37.7", "-122.4"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute positionals: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "Sunny with light winds" {
		t.Fatalf("positional tool output = %q, want %q", got, "Sunny with light winds")
	}
}

func TestToolsInspectOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tools", "weather", "get-forecast"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute inspect: %v\nstderr: %s", err, stderr.String())
	}
	output := stdout.String()
	for _, needle := range []string{"NAME", "USAGE", "ARGS", "--latitude <float>", "--longitude <float>"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("inspect output missing %q:\n%s", needle, output)
		}
	}
}

func TestDirectCommandMode(t *testing.T) {
	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tools", "--command", fixtureCommand(t)})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute tools --command: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "echo") {
		t.Fatalf("unexpected direct tools output: %s", stdout.String())
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "--command", fixtureCommand(t), "echo", "--input", `{"message":"direct"}`})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute tool --command: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "echo: direct" {
		t.Fatalf("direct tool output = %q, want %q", got, "echo: direct")
	}
}

func TestExposedCommandRewritesToTool(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: fixtureCommand(t), ExposeAs: []string{"wea"}}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "wea", ExposedCommandName: "wea"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(app.RewriteArgsForExposedMode(app.Invocation{ProgramName: "wea", ExposedCommandName: "wea"}, []string{"echo", "--input", `{"message":"hi"}`}))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "echo: hi" {
		t.Fatalf("exposed tool output = %q, want %q", got, "echo: hi")
	}
}

func fixtureCommand(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "testdata", "servers", "stdiofixture")
	return fmt.Sprintf("go run %q", fixtureDir)
}
