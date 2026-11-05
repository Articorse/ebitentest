package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"ebittest/worldgen"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	tiles         [data.ChunkSize * data.ChunkSize]data.TileEnum
	roofs         [data.ChunkSize * data.ChunkSize]data.RoofEnum
	promotedTiles map[utils.Vec2i]promotedTile
	features      map[utils.Vec2i]data.FeatureEnum
	// Only for damaged features
	featureHealth   map[utils.Vec2i]int
	sites           map[utils.Vec2i]*constructionSite
	terrainModified bool

	// The tiles of the features that give light, found when they're first asked for after the features change
	lights        []utils.Vec2i
	lightsIndexed bool
	// Every tile's walking speed, worked out when it's first asked for after a tile or feature changes, since paths ask
	// for it many times over
	walkSpeeds *[data.ChunkSize * data.ChunkSize]uint8

	Image *ebiten.Image
}

func (cc *ChunkContainer) newChunkData(pos utils.Vec2i) (*chunk, error) {
	c := chunk{
		tiles:    worldgen.ChunkTiles(cc.seed, pos),
		roofs:    worldgen.ChunkRoofs(cc.seed, pos),
		features: worldgen.ChunkFeatures(cc.seed, pos),
	}
	c.promotedTiles = make(map[utils.Vec2i]promotedTile)
	c.featureHealth = make(map[utils.Vec2i]int)
	c.sites = make(map[utils.Vec2i]*constructionSite)

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

func (x *chunk) getPromotedTiles() map[utils.Vec2i]promotedTile {
	return x.promotedTiles
}

// Returns false for tiles that can't be damaged, or that aren't in an active chunk. Returns what the tile drops, if
// this destroyed it.
func (cc *ChunkContainer) DamageTile(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	if drops, ok := cc.damageSite(t, damage); ok {
		return true, drops
	}

	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false, nil
	}

	idx := c.tileIndex(t)
	def, ok := cc.atlas[c.tiles[idx]]
	if !ok || def.maxHealth <= 0 {
		return false, nil
	}

	health := def.maxHealth
	if p, ok := c.promotedTiles[t]; ok {
		health = p.currentHealth
	}
	health -= damage

	if health > 0 {
		c.promotedTiles[t] = promotedTile{currentHealth: health}
		return true, nil
	}

	cc.setTile(c, t, def.destroyedInto)
	return true, def.drops
}

// Passable ground, loaded and without a feature or a construction site on it
func (cc *ChunkContainer) CanPlaceTile(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}
	if _, hasFeature := c.features[t]; hasFeature {
		return false
	}
	if _, hasSite := c.sites[t]; hasSite {
		return false
	}
	def, ok := cc.atlas[c.getTileDefId(t)]
	return ok && def.passable
}

// Returns false where a tile can't be placed, or the tile isn't in the atlas
func (cc *ChunkContainer) PlaceTile(t utils.Vec2i, tile data.TileEnum) bool {
	if _, ok := cc.atlas[tile]; !ok || !cc.CanPlaceTile(t) {
		return false
	}

	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	cc.setTile(c, t, tile)
	return true
}

// A new tile starts out undamaged. Only its own part of the chunk's image is drawn again, since baking a whole chunk's
// image is enough work to stall a frame.
func (cc *ChunkContainer) setTile(c *chunk, t utils.Vec2i, tile data.TileEnum) {
	idx := c.tileIndex(t)
	delete(c.promotedTiles, t)
	c.tiles[idx] = tile
	c.terrainModified = true
	c.walkSpeeds = nil

	if c.Image == nil {
		return
	}
	def, ok := cc.atlas[tile]
	if !ok {
		c.freeImage()
		return
	}
	drawTileInto(c.Image, idx, def.image)
}

func (cc *ChunkContainer) IsOpenToSky(t utils.Vec2i) bool {
	roof, ok := cc.RoofAt(t)
	return ok && roof == data.Roof_None
}

// False where no chunk is active
func (cc *ChunkContainer) RoofAt(t utils.Vec2i) (data.RoofEnum, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return data.Roof_None, false
	}
	return c.roofs[c.tileIndex(t)], true
}

// False where no chunk is active
func (cc *ChunkContainer) TileAt(t utils.Vec2i) (data.TileEnum, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return data.Tile_None, false
	}
	return c.getTileDefId(t), true
}

func (cc *ChunkContainer) CanDamageTile(t utils.Vec2i) bool {
	_, ok := cc.TileHealth(t)
	return ok
}

// A site with materials has its own health, which is what's hit
func (cc *ChunkContainer) TileHealth(t utils.Vec2i) (health int, ok bool) {
	if _, s, _, ok := cc.siteOn(t); ok && !s.isBlueprint() {
		return s.health, true
	}

	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0, false
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	if !ok || def.maxHealth <= 0 {
		return 0, false
	}

	if p, ok := c.promotedTiles[t]; ok {
		return p.currentHealth, true
	}
	return def.maxHealth, true
}
