package tiles

import (
	"sync"

	"gioui.org/op/paint"
)

type CombinedTileProvider struct {
	primary   TileProvider
	fallback  TileProvider
	loading   map[string]bool
	loadingMu sync.RWMutex
}

func NewCombinedTileProvider(primary, fallback TileProvider) *CombinedTileProvider {
	return &CombinedTileProvider{
		primary:  primary,
		fallback: fallback,
		loading:  make(map[string]bool),
	}
}

func (p *CombinedTileProvider) GetTile(tile Tile) (*paint.ImageOp, error) {
	key := getTileKey(tile)

	// Check if we're already loading this tile
	p.loadingMu.RLock()
	isLoading := p.loading[key]
	p.loadingMu.RUnlock()

	if isLoading {
		// Return fallback tile while loading
		return p.fallback.GetTile(tile)
	}

	// Mark tile as loading
	p.loadingMu.Lock()
	p.loading[key] = true
	p.loadingMu.Unlock()

	// Try to get the primary tile
	img, err := p.primary.GetTile(tile)

	// Clear loading status
	p.loadingMu.Lock()
	delete(p.loading, key)
	p.loadingMu.Unlock()

	if err != nil {
		// If primary fails, return fallback
		return p.fallback.GetTile(tile)
	}

	return img, nil
}
