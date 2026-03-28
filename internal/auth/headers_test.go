package auth

import (
	"path/filepath"
	"testing"

	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestHeadersForServerBearerEnv(t *testing.T) {
	t.Setenv("ACME_TOKEN", "secret")
	headers, err := HeadersForServer(nil, &config.Server{BearerEnv: "ACME_TOKEN"})
	if err != nil {
		t.Fatalf("HeadersForServer: %v", err)
	}
	if headers["Authorization"] != "Bearer secret" {
		t.Fatalf("Authorization = %q", headers["Authorization"])
	}
}

func TestHeadersForServerOAuthToken(t *testing.T) {
	store := &Store{fileDir: filepath.Join(t.TempDir(), "tokens")}
	server := &config.Server{Name: "notion", Auth: "oauth"}
	if err := store.Save(TokenKey(server), &Token{AccessToken: "oauth-token"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	headers, err := HeadersForServer(store, server)
	if err != nil {
		t.Fatalf("HeadersForServer: %v", err)
	}
	if headers["Authorization"] != "Bearer oauth-token" {
		t.Fatalf("Authorization = %q", headers["Authorization"])
	}
}
