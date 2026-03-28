package cli

import (
	"github.com/maximerivest/mcp2cli/internal/cache"
	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
	"github.com/maximerivest/mcp2cli/internal/naming"
)

func cacheMetadata(state *State, server *config.Server, mutate func(*cache.Metadata)) {
	store, err := state.MetadataStore()
	if err != nil || store == nil {
		return
	}
	metadata, err := store.Load(server)
	if err != nil || metadata == nil {
		metadata = &cache.Metadata{}
	}
	mutate(metadata)
	_ = store.Save(server, metadata)
}

// cachedToolSchema returns a tool from the metadata cache if available.
// This allows skipping the tools/list round trip on tool invocation.
func cachedToolSchema(state *State, server *config.Server, toolName string) *types.Tool {
	store, err := state.MetadataStore()
	if err != nil || store == nil {
		return nil
	}
	metadata, err := store.Load(server)
	if err != nil || metadata == nil || len(metadata.Tools) == 0 {
		return nil
	}
	requestedCLI := naming.ToKebabCase(toolName)
	for _, tool := range metadata.Tools {
		if tool.Name == toolName || naming.ToKebabCase(tool.Name) == requestedCLI {
			return &tool
		}
	}
	return nil
}
