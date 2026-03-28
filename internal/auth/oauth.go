package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/pkg/browser"
)

// OpenURL opens an auth URL in the browser. Tests may replace it.
var OpenURL = browser.OpenURL

// LoginOAuth performs an OAuth authorization-code flow and stores the token.
func LoginOAuth(ctx context.Context, store *Store, server *config.Server) (*Token, error) {
	cfg, err := oauthConfigForServer(server)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "listen for OAuth callback")
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://%s/callback", listener.Addr().String())
	state, err := randomState()
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "generate OAuth state")
	}

	callbackCh := make(chan string, 1)
	errCh := make(chan error, 1)
	serverHTTP := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			errCh <- exitcode.New(exitcode.Auth, "oauth state mismatch")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- exitcode.New(exitcode.Auth, "oauth callback did not include a code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprintln(w, "Authentication complete. You can close this window.")
		callbackCh <- code
	})}
	go func() { _ = serverHTTP.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = serverHTTP.Shutdown(shutdownCtx)
	}()

	authURL := cfg.AuthorizeURL
	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	if len(cfg.Scopes) > 0 {
		query.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	authURL.RawQuery = query.Encode()

	if err := OpenURL(authURL.String()); err != nil {
		fmt.Fprintf(os.Stderr, "Open this URL to authenticate:\n%s\n", authURL.String())
	}

	var code string
	select {
	case code = <-callbackCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, exitcode.Wrap(exitcode.Auth, ctx.Err(), "oauth login timed out")
	}

	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("client_id", cfg.ClientID)
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "build token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Auth, err, "exchange oauth code")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, exitcode.Newf(exitcode.Auth, "oauth token exchange failed with HTTP %d", resp.StatusCode)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "decode oauth token")
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return nil, exitcode.New(exitcode.Auth, "oauth token response did not include an access token")
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}

	if store != nil {
		if err := store.Save(TokenKey(server), &token); err != nil {
			return nil, err
		}
	}
	return &token, nil
}

type oauthConfig struct {
	AuthorizeURL *url.URL
	TokenURL     *url.URL
	ClientID     string
	Scopes       []string
}

func oauthConfigForServer(server *config.Server) (*oauthConfig, error) {
	if server == nil || strings.TrimSpace(server.URL) == "" {
		return nil, exitcode.New(exitcode.Config, "oauth requires a remote server URL")
	}
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Config, err, "parse server URL")
	}
	resolve := func(value, fallback string) (*url.URL, error) {
		if strings.TrimSpace(value) != "" {
			parsed, err := url.Parse(value)
			if err != nil {
				return nil, exitcode.Wrap(exitcode.Config, err, "parse oauth endpoint URL")
			}
			if parsed.IsAbs() {
				return parsed, nil
			}
			return baseURL.ResolveReference(parsed), nil
		}
		parsed, _ := url.Parse(fallback)
		return baseURL.ResolveReference(parsed), nil
	}
	authorizeURL, err := resolve(server.OAuthAuthorizeURL, "/oauth/authorize")
	if err != nil {
		return nil, err
	}
	tokenURL, err := resolve(server.OAuthTokenURL, "/oauth/token")
	if err != nil {
		return nil, err
	}
	clientID := server.OAuthClientID
	if clientID == "" {
		clientID = "mcp2cli"
	}
	return &oauthConfig{AuthorizeURL: authorizeURL, TokenURL: tokenURL, ClientID: clientID, Scopes: append([]string(nil), server.OAuthScopes...)}, nil
}

func randomState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
