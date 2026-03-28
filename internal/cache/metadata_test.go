package cache

import (
	"path/filepath"
	"testing"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

func TestStoreSaveLoad(t *testing.T) {
	store := NewStore(config.Paths{MetadataCacheDir: filepath.Join(t.TempDir(), "metadata")})
	server := &config.Server{Name: "weather"}
	metadata := &Metadata{Tools: []types.Tool{{Name: "echo"}}}
	if err := store.Save(server, metadata); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load(server)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Tools) != 1 || loaded.Tools[0].Name != "echo" {
		t.Fatalf("loaded tools = %#v", loaded.Tools)
	}
}
