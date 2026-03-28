package cli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestDispatchShellLine(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	state, err := newState(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("newState: %v", err)
	}
	repo, err := state.Repo()
	if err != nil {
		t.Fatalf("Repo: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "weather", &config.Server{Command: shellFixtureCommand(t)}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stdout := &bytes.Buffer{}
	resolved, session, err := openSession(state, metadataConnectionOptions{ExplicitName: "weather"}, ctx, strings.NewReader(""), stdout)
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	defer func() { _ = session.Close() }()

	env, err := newShellEnv(state, resolved, session, "auto", stdout, ctx)
	if err != nil {
		t.Fatalf("newShellEnv: %v", err)
	}

	if err := dispatchShellLine(env, "get-forecast 1 2"); err != nil {
		t.Fatalf("dispatch tool: %v", err)
	}
	if !strings.Contains(stdout.String(), "Sunny with light winds") {
		t.Fatalf("tool output: %s", stdout.String())
	}

	stdout.Reset()
	if err := dispatchShellLine(env, "resource api-docs"); err != nil {
		t.Fatalf("dispatch resource: %v", err)
	}
	if !strings.Contains(stdout.String(), "API docs for the weather service") {
		t.Fatalf("resource output: %s", stdout.String())
	}

	stdout.Reset()
	if err := dispatchShellLine(env, "prompt review-code --code 'x <- 1' --focus api"); err != nil {
		t.Fatalf("dispatch prompt: %v", err)
	}
	if !strings.Contains(stdout.String(), "Review this code with a focus on api") {
		t.Fatalf("prompt output: %s", stdout.String())
	}

	stdout.Reset()
	if err := dispatchShellLine(env, "set output json"); err != nil {
		t.Fatalf("dispatch set output: %v", err)
	}
	if env.output != "json" {
		t.Fatalf("env.output = %q", env.output)
	}
}

func shellFixtureCommand(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "testdata", "servers", "stdiofixture")
	return fmt.Sprintf("go run %q", fixtureDir)
}
