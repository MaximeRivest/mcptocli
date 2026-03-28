package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
)

func TestLoginOAuth(t *testing.T) {
	var redirectSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/authorize":
			redirectSeen = true
			redirect := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, fmt.Sprintf("%s?code=test-code&state=%s", redirect, state), http.StatusFound)
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"oauth-token","token_type":"Bearer"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &Store{fileDir: filepath.Join(t.TempDir(), "tokens")}
	oldOpenURL := OpenURL
	OpenURL = func(raw string) error {
		resp, err := http.Get(raw)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}
	defer func() { OpenURL = oldOpenURL }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token, err := LoginOAuth(ctx, store, &config.Server{Name: "notion", URL: server.URL, Auth: "oauth"})
	if err != nil {
		t.Fatalf("LoginOAuth: %v", err)
	}
	if !redirectSeen {
		t.Fatal("authorize endpoint was not called")
	}
	if token.AccessToken != "oauth-token" {
		t.Fatalf("AccessToken = %q", token.AccessToken)
	}
	loaded, err := store.Load(TokenKey(&config.Server{Name: "notion", URL: server.URL, Auth: "oauth"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AccessToken != "oauth-token" {
		t.Fatalf("stored token = %q", loaded.AccessToken)
	}
}
