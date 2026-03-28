package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/app"
	"github.com/maximerivest/mcp2cli/internal/auth"
	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestRemoteHTTPBearerFlow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("ACME_TOKEN", "secret")

	server := newRemoteFixtureServer(t, "secret")
	defer server.Close()

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tools", "--url", server.URL + "/mcp", "--bearer-env", "ACME_TOKEN"})
	if err := root.Execute(); err != nil {
		t.Fatalf("tools --url: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "echo") {
		t.Fatalf("unexpected tools output: %s", stdout.String())
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "--url", server.URL + "/mcp", "--bearer-env", "ACME_TOKEN", "echo", "--message", "remote"})
	if err := root.Execute(); err != nil {
		t.Fatalf("tool --url: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "echo: remote" {
		t.Fatalf("remote tool output = %q", got)
	}
}

func TestLoginOAuthAndRemoteCall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	server := newRemoteFixtureServer(t, "oauth-token")
	defer server.Close()

	repo, err := config.NewRepository("")
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	if err := repo.UpsertServer(config.SourceGlobal, "notion", &config.Server{URL: server.URL + "/mcp", Auth: "oauth"}); err != nil {
		t.Fatalf("UpsertServer: %v", err)
	}

	oldOpenURL := auth.OpenURL
	auth.OpenURL = func(raw string) error {
		resp, err := http.Get(raw)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}
	defer func() { auth.OpenURL = oldOpenURL }()

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"login", "notion", "--timeout", (2 * time.Second).String()})
	if err := root.Execute(); err != nil {
		t.Fatalf("login: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "login successful") {
		t.Fatalf("unexpected login output: %s", stdout.String())
	}

	root, err = NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"tool", "notion", "echo", "--message", "oauth"})
	if err := root.Execute(); err != nil {
		t.Fatalf("remote oauth tool: %v\nstderr: %s", err, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "echo: oauth" {
		t.Fatalf("oauth tool output = %q", got)
	}
}

func TestDoctorRemote(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("ACME_TOKEN", "secret")

	server := newRemoteFixtureServer(t, "secret")
	defer server.Close()

	root, err := NewRootCommand(Options{Version: "dev", Invocation: app.Invocation{ProgramName: "mcp2cli"}})
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"doctor", "--url", server.URL + "/mcp", "--bearer-env", "ACME_TOKEN"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor: %v\nstderr: %s", err, stderr.String())
	}
	for _, needle := range []string{"resolve", "auth", "connect", "tools"} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("doctor output missing %q:\n%s", needle, stdout.String())
		}
	}
}

func newRemoteFixtureServer(t *testing.T, requiredToken string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/authorize":
			redirect := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, fmt.Sprintf("%s?code=test-code&state=%s", redirect, state), http.StatusFound)
			return
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"access_token":%q,"token_type":"Bearer"}`, requiredToken)))
			return
		case "/mcp":
			if requiredToken != "" {
				if got := r.Header.Get("Authorization"); got != "Bearer "+requiredToken {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
					return
				}
			}
			defer r.Body.Close()
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			method, _ := payload["method"].(string)
			id := payload["id"]
			w.Header().Set("Content-Type", "application/json")
			switch method {
			case "initialize":
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "serverInfo": map[string]any{"name": "remote-fixture", "version": "0.1.0"}}})
			case "notifications/initialized":
				_, _ = w.Write([]byte(`{}`))
			case "tools/list":
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "echo", "description": "Echo a message back", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"message": map[string]any{"type": "string", "description": "Message to echo"}}, "required": []string{"message"}}}}}})
			case "tools/call":
				params, _ := payload["params"].(map[string]any)
				arguments, _ := params["arguments"].(map[string]any)
				message, _ := arguments["message"].(string)
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"content": []map[string]any{{"type": "text", "text": "echo: " + message}}, "structuredContent": map[string]any{"message": message}}})
			default:
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": "method not found"}})
			}
		default:
			http.NotFound(w, r)
		}
	}))
}
