package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	tiles         [data.ChunkSize * data.ChunkSize]data.TileEnum
	promotedTiles map[utils.Vec2i]promotedTile
	// Generated again whenever the chunk is loaded rather than saved, since nothing can change them yet
	features        map[utils.Vec2i]data.FeatureEnum
	terrainModified bool

	Image *ebiten.Image
}

func (cc *ChunkContainer) newChunkData(pos utils.Vec2i) (*chunk, error) {
	c := chunk{tiles: worldgen.ChunkTiles(cc.seed, pos), features: worldgen.ChunkFeatures(cc.seed, pos)}
	c.promotedTiles = make(map[utils.Vec2i]promotedTile)

	return &c, nil
}

func (x *chunk) getTileDefId(cellKey utils.Vec2i) data.TileEnum {
	return x.tiles[x.tileIndex(cellKey)]
}

func (x *chunk) tileIndex(cellKey utils.Vec2i) int {
	localX := ((cellKey.X % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	localY := ((cellKey.Y % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	return localY*data.ChunkSize + localX
}

// The ground under a feature can't be damaged while the feature stands on it
func (x *chunk) protectsGround(t utils.Vec2i) bool {
	_, ok := x.features[t]
	return ok
}

func (x *chunk) getPromotedTiles() map[utils.Vec2i]promotedTile {
	return x.promotedTiles
}

// Returns false for tiles that can't be damaged, or that aren't in an active chunk
func (cc *ChunkContainer) DamageTile(t utils.Vec2i, damage int) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}

	idx := c.tileIndex(t)
	def, ok := cc.atlas[c.tiles[idx]]
	if !ok || def.maxHealth <= 0 || c.protectsGround(t) {
		return false
	}

	health := def.maxHealth
	if p, ok := c.promotedTiles[t]; ok {
		health = p.currentHealth
	}
	health -= damage

	if health > 0 {
		c.promotedTiles[t] = promotedTile{currentHealth: health}
		return true
	}

	delete(c.promotedTiles, t)
	c.tiles[idx] = def.destroyedInto
	c.terrainModified = true
	c.freeImage()
	return true
}

func (cc *ChunkContainer) CanDamageTile(t utils.Vec2i) bool {
	_, ok := cc.TileHealth(t)
	return ok
}

func (cc *ChunkContainer) TileHealth(t utils.Vec2i) (health int, ok bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0, false
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	if !ok || def.maxHealth <= 0 || c.protectsGround(t) {
		return 0, false
	}

	if p, ok := c.promotedTiles[t]; ok {
		return p.currentHealth, true
	}
	return def.maxHealth, true
}
