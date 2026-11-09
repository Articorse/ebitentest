package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"ebittest/worldgen"

	"github.com/hajimehoshi/ebiten/v2"
)

type chunk struct {
	tiles [data.TileLayerCount][data.ChunkSize * data.ChunkSize]data.TileEnum
	// Worked out from tiles whenever one changes, and once the chunk is loaded
	props         [data.ChunkSize * data.ChunkSize]tileProps
	roofs         [data.ChunkSize * data.ChunkSize]data.RoofEnum
	promotedTiles map[utils.Vec2i]promotedTile
	features      map[utils.Vec2i]data.FeatureEnum
	// Only for damaged features
	featureHealth   map[utils.Vec2i]int
	sites           map[utils.Vec2i]*constructionSite
	terrainModified bool
	// One bit per tile for each area, nil for a chunk with none of it, since most have none
	areas [data.AreaCount]*[data.ChunkSize * data.ChunkSize / 64]uint64

	// The tiles of the features that give light, found when they're first asked for after the features change
	lights        []utils.Vec2i
	lightsIndexed bool
	// Every tile's walking speed, worked out when it's first asked for after a tile or feature changes, since paths ask
	// for it many times over
	walkSpeeds *[data.ChunkSize * data.ChunkSize]uint8
	// Which of the chunk's regions each tile is in, -1 where it isn't walked on. Nil until it's asked for after a tile or
	// feature changes.
	regionOf *[data.ChunkSize * data.ChunkSize]int32
	regions  []chunkRegion
	// Lazily for each size of body that asks, until a tile or feature here or in a chunk next to it changes
	bodyFits map[bodyBox]*[data.ChunkSize * data.ChunkSize]uint8
	// Nil until it's asked for after the features change
	opaqueFeatures map[utils.Vec2i]bool
	// Whose each door is, who it opens for
	doorFactions map[utils.Vec2i]data.FactionEnum

	Image *ebiten.Image
	// What lies on each tile
	ground map[utils.Vec2i]ecs.GroundStack
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
	c.ground = make(map[utils.Vec2i]ecs.GroundStack)
	c.doorFactions = make(map[utils.Vec2i]data.FactionEnum)

	return &c, nil
}

func (x *chunk) tilesAt(t utils.Vec2i) data.TileLayers {
	var tiles data.TileLayers
	idx := x.tileIndex(t)
	for l := range tiles {
		tiles[l] = x.tiles[l][idx]
	}
	return tiles
}

func (x *chunk) propsAt(t utils.Vec2i) tileProps {
	return x.props[x.tileIndex(t)]
}

func (x *chunk) tileIndex(cellKey utils.Vec2i) int {
	localX := ((cellKey.X % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	localY := ((cellKey.Y % data.ChunkSize) + data.ChunkSize) % data.ChunkSize
	return localY*data.ChunkSize + localX
}

func (x *chunk) getPromotedTiles() map[utils.Vec2i]promotedTile {
	return x.promotedTiles
}

// Only the highest layer is hit, and destroying it uncovers the one under it. Returns false for tiles that can't be
// damaged, or that aren't in an active chunk. Returns what the tile drops, if this destroyed it.
func (cc *ChunkContainer) DamageTile(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	if drops, ok := cc.damageSite(t, damage); ok {
		return true, drops
	}

	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false, nil
	}

	layer := c.tilesAt(t).TopLayer()
	def, ok := cc.atlas[c.tiles[layer][c.tileIndex(t)]]
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

	cc.setTile(c, t, layer, data.Tile_None)
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
	return c.propsAt(t).passable
}

// On the tile's own layer, in place of what was there. Returns false where a tile can't be placed, or the tile isn't in
// the atlas.
func (cc *ChunkContainer) PlaceTile(t utils.Vec2i, tile data.TileEnum) bool {
	def, ok := cc.atlas[tile]
	if !ok || !cc.CanPlaceTile(t) {
		return false
	}

	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	cc.setTile(c, t, def.layer, tile)
	return true
}

// Puts the tile on the layer, or takes what's there off it with Tile_None. A wall clears the cover under it, rather than
// leaving it to come back once the wall's gone. The tile's top starts out undamaged. Only its own part of the chunk's
// image is drawn again, since baking a whole chunk's image is enough work to stall a frame.
func (cc *ChunkContainer) setTile(c *chunk, t utils.Vec2i, layer data.TileLayerEnum, tile data.TileEnum) {
	idx := c.tileIndex(t)
	replaced := []data.TileLayerEnum{layer}
	if layer == data.TileLayer_Wall && tile != data.Tile_None {
		replaced = append(replaced, data.TileLayer_Cover)
	}
	for _, l := range replaced {
		if old := c.tiles[l][idx]; old != data.Tile_None && cc.atlas[old].harvestable {
			cc.noteHarvested(t)
		}
		c.tiles[l][idx] = data.Tile_None
	}
	c.tiles[layer][idx] = tile
	c.props[idx] = cc.combine(c.tilesAt(t))
	delete(c.promotedTiles, t)
	c.terrainModified = true
	cc.sightVersion++
	c.walkSpeeds = nil
	c.regionOf, cc.regionGraph = nil, nil
	cc.forgetBodyFits(utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)})

	cc.redrawAround(t)
	cc.turnDoorsAround(t)
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
func (cc *ChunkContainer) TileAt(t utils.Vec2i) (data.TileLayers, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return data.TileLayers{}, false
	}
	return c.tilesAt(t), true
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

	def, ok := cc.atlas[c.tilesAt(t).Top()]
	if !ok || def.maxHealth <= 0 {
		return 0, false
	}

	if p, ok := c.promotedTiles[t]; ok {
		return p.currentHealth, true
	}
	return def.maxHealth, true
}
