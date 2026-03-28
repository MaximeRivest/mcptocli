package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

// Metadata stores cached MCP metadata for one server.
type Metadata struct {
	SavedAt   time.Time        `json:"savedAt"`
	Tools     []types.Tool     `json:"tools,omitempty"`
	Resources []types.Resource `json:"resources,omitempty"`
	Prompts   []types.Prompt   `json:"prompts,omitempty"`
}

// Store persists metadata in the cache directory.
type Store struct {
	dir string
}

// NewStore creates a metadata store from config paths.
func NewStore(paths config.Paths) *Store {
	return &Store{dir: paths.MetadataCacheDir}
}

// Save writes metadata for a server fingerprint.
func (s *Store) Save(server *config.Server, metadata *Metadata) error {
	if s == nil || metadata == nil {
		return nil
	}
	if s.dir == "" {
		return nil
	}
	metadata.SavedAt = time.Now().UTC()
	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "marshal metadata cache")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "create metadata cache directory")
	}
	path := filepath.Join(s.dir, cacheKey(server)+".json")
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "write metadata cache")
	}
	return nil
}

// Load reads cached metadata if available.
func (s *Store) Load(server *config.Server) (*Metadata, error) {
	if s == nil || s.dir == "" {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(s.dir, cacheKey(server)+".json")
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var metadata Metadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return nil, fmt.Errorf("decode metadata cache: %w", err)
	}
	return &metadata, nil
}

func cacheKey(server *config.Server) string {
	if server == nil {
		return "unknown"
	}
	base := server.Name
	if base == "" || base == "(direct)" {
		base = server.URL
		if base == "" {
			base = server.Command
		}
	}
	hash := sha256.Sum256([]byte(base))
	return hex.EncodeToString(hash[:12])
}
