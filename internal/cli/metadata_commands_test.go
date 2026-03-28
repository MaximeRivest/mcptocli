package cli

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestResourcesAndPromptsCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: metadataFixtureCommand(t)}); err != nil {
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
	root.SetArgs([]string{"resources", "weather"})
	if err := root.Execute(); err != nil {
		t.Fatalf("resources: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "api-docs") || !strings.Contains(stdout.String(), "current-status") {
		t.Fatalf("unexpected resources output: %s", stdout.String())
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"resource", "weather", "api-docs"})
	if err := root.Execute(); err != nil {
		t.Fatalf("resource: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "API docs for the weather service" {
		t.Fatalf("resource output = %q", got)
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"prompts", "weather", "review-code"})
	if err := root.Execute(); err != nil {
		t.Fatalf("prompts inspect: %v\nstderr: %s", err, stderr.String())
	}
	for _, needle := range []string{"NAME", "ARGS", "--code string"} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("prompt inspect missing %q:\n%s", needle, stdout.String())
		}
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"prompt", "weather", "review-code", "--code", "fmt.Println(1)", "--focus", "api"})
	if err := root.Execute(); err != nil {
		t.Fatalf("prompt render: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Review this code with a focus on api") {
		t.Fatalf("unexpected prompt output: %s", stdout.String())
	}
}

func metadataFixtureCommand(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "testdata", "servers", "stdiofixture")
	return fmt.Sprintf("go run %q", fixtureDir)
}
