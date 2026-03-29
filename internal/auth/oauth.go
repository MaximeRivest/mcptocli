package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maximerivest/mcptocli/internal/config"
	"github.com/maximerivest/mcptocli/internal/exitcode"
	"github.com/pkg/browser"
)

// OpenURL opens an auth URL in the browser. Tests may replace it.
var OpenURL = browser.OpenURL

// LoginOAuth performs an OAuth authorization-code flow and stores the token.
// Supports RFC 8414 discovery, dynamic client registration (RFC 7591), and PKCE (RFC 7636).
func LoginOAuth(ctx context.Context, store *Store, server *config.Server) (*Token, error) {
	cfg, err := oauthConfigForServer(ctx, server)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "listen for OAuth callback")
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://%s/callback", listener.Addr().String())

	// Dynamic client registration if no client_id configured and registration endpoint available
	if cfg.ClientID == "" && cfg.RegistrationURL != nil {
		clientID, err := dynamicClientRegistration(ctx, cfg.RegistrationURL, redirectURI)
		if err != nil {
			return nil, exitcode.Wrap(exitcode.Auth, err, "dynamic client registration")
		}
		cfg.ClientID = clientID
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "mcptocli"
	}

	state, err := randomState()
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "generate OAuth state")
	}

	// Generate PKCE challenge
	codeVerifier, codeChallenge, err := generatePKCE()
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "generate PKCE challenge")
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

	authURL := *cfg.AuthorizeURL
	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
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
	values.Set("code_verifier", codeVerifier)

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
		body, _ := io.ReadAll(resp.Body)
		return nil, exitcode.Newf(exitcode.Auth, "oauth token exchange failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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

// oauthConfig holds the resolved OAuth endpoints.
type oauthConfig struct {
	AuthorizeURL    *url.URL
	TokenURL        *url.URL
	RegistrationURL *url.URL
	ClientID        string
	Scopes          []string
}

// oauthDiscovery represents the RFC 8414 OAuth Authorization Server Metadata.
type oauthDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
}

// discoverOAuthConfig tries RFC 8414 discovery for the server's origin.
func discoverOAuthConfig(ctx context.Context, serverURL *url.URL) *oauthDiscovery {
	// Try /.well-known/oauth-authorization-server at the origin
	wellKnown := &url.URL{
		Scheme: serverURL.Scheme,
		Host:   serverURL.Host,
		Path:   "/.well-known/oauth-authorization-server",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown.String(), nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var disc oauthDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil
	}
	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" {
		return nil
	}
	return &disc
}

func oauthConfigForServer(ctx context.Context, server *config.Server) (*oauthConfig, error) {
	if server == nil || strings.TrimSpace(server.URL) == "" {
		return nil, exitcode.New(exitcode.Config, "oauth requires a remote server URL")
	}
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Config, err, "parse server URL")
	}

	// Try RFC 8414 discovery first
	disc := discoverOAuthConfig(ctx, baseURL)

	resolve := func(explicit, discovered, fallback string) (*url.URL, error) {
		// 1. Explicit config wins
		if strings.TrimSpace(explicit) != "" {
			parsed, err := url.Parse(explicit)
			if err != nil {
				return nil, exitcode.Wrap(exitcode.Config, err, "parse oauth endpoint URL")
			}
			if parsed.IsAbs() {
				return parsed, nil
			}
			return baseURL.ResolveReference(parsed), nil
		}
		// 2. Discovered endpoint
		if strings.TrimSpace(discovered) != "" {
			parsed, err := url.Parse(discovered)
			if err == nil && parsed.IsAbs() {
				return parsed, nil
			}
		}
		// 3. Fallback
		parsed, _ := url.Parse(fallback)
		return baseURL.ResolveReference(parsed), nil
	}

	var discAuth, discToken, discReg string
	if disc != nil {
		discAuth = disc.AuthorizationEndpoint
		discToken = disc.TokenEndpoint
		discReg = disc.RegistrationEndpoint
	}

	authorizeURL, err := resolve(server.OAuthAuthorizeURL, discAuth, "/oauth/authorize")
	if err != nil {
		return nil, err
	}
	tokenURL, err := resolve(server.OAuthTokenURL, discToken, "/oauth/token")
	if err != nil {
		return nil, err
	}

	var registrationURL *url.URL
	if discReg != "" {
		registrationURL, _ = url.Parse(discReg)
	}

	clientID := server.OAuthClientID

	return &oauthConfig{
		AuthorizeURL:    authorizeURL,
		TokenURL:        tokenURL,
		RegistrationURL: registrationURL,
		ClientID:        clientID,
		Scopes:          append([]string(nil), server.OAuthScopes...),
	}, nil
}

// dynamicClientRegistration registers a new OAuth client per RFC 7591.
func dynamicClientRegistration(ctx context.Context, registrationURL *url.URL, redirectURI string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"client_name":   "mcptocli",
		"redirect_uris": []string{redirectURI},
		"grant_types":   []string{"authorization_code"},
		"response_types": []string{"code"},
		"token_endpoint_auth_method": "none",
	})
	if err != nil {
		return "", fmt.Errorf("marshal registration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationURL.String(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("registration failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode registration response: %w", err)
	}
	if result.ClientID == "" {
		return "", fmt.Errorf("registration response did not include client_id")
	}
	return result.ClientID, nil
}

// generatePKCE creates a code_verifier and S256 code_challenge per RFC 7636.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

func randomState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
