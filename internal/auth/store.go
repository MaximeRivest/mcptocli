package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
)

// Token represents an access token stored for remote auth.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// Store persists tokens using keyring with file fallback.
type Store struct {
	keyring keyring.Keyring
	fileDir string
}

// NewStore creates a token store using repository paths.
func NewStore(paths config.Paths) *Store {
	store := &Store{fileDir: paths.TokenDir}
	if os.Getenv("MCP2CLI_USE_SYSTEM_KEYRING") != "1" {
		return store
	}
	type result struct {
		keyring keyring.Keyring
		err     error
	}
	ch := make(chan result, 1)
	go func() {
		kr, err := keyring.Open(keyring.Config{ServiceName: "mcp2cli", FileDir: paths.TokenDir})
		ch <- result{keyring: kr, err: err}
	}()
	select {
	case opened := <-ch:
		if opened.err == nil {
			store.keyring = opened.keyring
		}
	case <-time.After(200 * time.Millisecond):
		// Fall back to file storage if the system keyring is slow or unavailable.
	}
	return store
}

// TokenKey derives a stable token key for a server.
func TokenKey(server *config.Server) string {
	if server == nil {
		return "unknown"
	}
	if server.Name != "" && server.Name != "(direct)" {
		return "server-" + server.Name
	}
	base := server.URL
	if base == "" {
		base = server.Command
	}
	hash := sha256.Sum256([]byte(base))
	return "server-" + hex.EncodeToString(hash[:8])
}

// Save stores a token.
func (s *Store) Save(key string, token *Token) error {
	if token == nil {
		return exitcode.New(exitcode.Internal, "token cannot be nil")
	}
	data, err := json.Marshal(token)
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "marshal token")
	}
	if s.keyring != nil {
		if err := s.keyring.Set(keyring.Item{Key: key, Data: data}); err == nil {
			return nil
		}
	}
	if s.fileDir == "" {
		return exitcode.New(exitcode.Internal, "token storage is not configured")
	}
	if err := os.MkdirAll(s.fileDir, 0o700); err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "create token directory")
	}
	path := filepath.Join(s.fileDir, sanitizeKey(key)+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "write token file")
	}
	return nil
}

// Load retrieves a token.
func (s *Store) Load(key string) (*Token, error) {
	if s.keyring != nil {
		item, err := s.keyring.Get(key)
		if err == nil {
			var token Token
			if err := json.Unmarshal(item.Data, &token); err != nil {
				return nil, exitcode.Wrap(exitcode.Internal, err, "decode stored token")
			}
			return &token, nil
		}
	}
	if s.fileDir == "" {
		return nil, exitcode.New(exitcode.Auth, "token not found")
	}
	path := filepath.Join(s.fileDir, sanitizeKey(key)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Auth, err, "token not found")
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "decode stored token")
	}
	return &token, nil
}

func sanitizeKey(key string) string {
	replacer := strings.NewReplacer("/", "-", string(filepath.Separator), "-", ":", "-")
	return replacer.Replace(key)
}
