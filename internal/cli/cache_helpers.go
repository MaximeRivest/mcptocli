package cli

import (
	"github.com/maximerivest/mcp2cli/internal/cache"
	"github.com/maximerivest/mcp2cli/internal/config"
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
