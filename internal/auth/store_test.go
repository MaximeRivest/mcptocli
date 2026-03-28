package auth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestStoreSaveLoadFallback(t *testing.T) {
	paths := config.Paths{TokenDir: filepath.Join(t.TempDir(), "tokens")}
	store := &Store{fileDir: paths.TokenDir}
	token := &Token{AccessToken: "abc123", TokenType: "Bearer", Expiry: time.Now().UTC().Round(0)}
	if err := store.Save("server-weather", token); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load("server-weather")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AccessToken != token.AccessToken {
		t.Fatalf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}
}
