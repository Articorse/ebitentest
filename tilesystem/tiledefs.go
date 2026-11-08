package tilesystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

type tileDef struct {
	image *ebiten.Image
	layer data.TileLayerEnum
	// Not positive for tiles that can't be damaged. A destroyed one is taken off its layer, uncovering what's under it.
	maxHealth int
	passable  bool
	// An impassable tile that's water is on Layer_Water rather than Layer_Wall, so that it stops walkers but not
	// what flies over it
	water  bool
	opaque bool
	// Construction sites can be placed on it
	buildable bool
	// How much slower walking over it is, which only paths take into account for now
	slowdownPercent int
	// Nil for tiles that drop nothing when destroyed
	drops *ecs.DropTable
	// Gives what's gathered on the harvest area, which keeps the area there for as long as it's left
	harvestable bool
	// Nil for a tile drawn whole, as image. Otherwise its 16 dual-grid cells, laid out as dualGridCells says, which are
	// drawn over those of the tiles on its layer with a lower drawOrder where they meet.
	dualGrid  []*ebiten.Image
	drawOrder int
}

type promotedTile struct {
	currentHealth int
}

type tileDefDTO struct {
	Id         uint16 `json:"id"`
	ImageAsset string `json:"imageAsset"`
	Layer      string `json:"layer"`
	// Not positive for tiles that can't be damaged
	MaxHealth       int              `json:"maxHealth"`
	Passable        bool             `json:"passable"`
	Water           bool             `json:"water"`
	Opaque          bool             `json:"opaque"`
	Buildable       bool             `json:"buildable"`
	SlowdownPercent int              `json:"slowdownPercent"`
	Drops           []dropOutcomeDTO `json:"drops"`
	Harvestable     bool             `json:"harvestable"`
	// Given for a tile drawn on the dual grid, from a sheet of 16 cells
	DrawOrder *int `json:"drawOrder"`
}

func dtosToDefsMap(dtos []tileDefDTO, items *ecs.ItemAtlas, assets *assetmanager.AssetManager) (map[data.TileEnum]tileDef, error) {
	tileDefs := make(map[data.TileEnum]tileDef, len(dtos))
	for _, dto := range dtos {
		asset, err := assets.GetAsset(common.ImageAssetTag(dto.ImageAsset))
		if err != nil {
			return nil, fmt.Errorf("tile %d's image: %w", dto.Id, err)
		}
		img := asset.Frames[0]
		layer, ok := tileLayers[dto.Layer]
		if !ok {
			return nil, fmt.Errorf("tile %d is on layer %q, want ground, foundation, cover or wall", dto.Id, dto.Layer)
		}
		if dto.Id == uint16(data.Tile_None) {
			return nil, fmt.Errorf("tile %d is what an empty layer holds, so it can't be defined", dto.Id)
		}

		def := tileDef{
			layer:     layer,
			image:     img,
			maxHealth: dto.MaxHealth,
			passable:  dto.Passable,
			water:     dto.Water,
			opaque:    dto.Opaque,
			buildable: dto.Buildable,

			slowdownPercent: dto.SlowdownPercent,
			harvestable:     dto.Harvestable,
		}
		if dto.DrawOrder != nil {
			if len(asset.Frames) != len(dualGridCells) {
				return nil, fmt.Errorf("tile %d is drawn on the dual grid, but its image has %d frames rather than %d", dto.Id, len(asset.Frames), len(dualGridCells))
			}
			def.dualGrid, def.drawOrder = asset.Frames, *dto.DrawOrder
		}
		if dto.SlowdownPercent < 0 || dto.SlowdownPercent >= 100 {
			return nil, fmt.Errorf("tile %d slows walking by %d%%, want 0 to 99, or the tile impassable", dto.Id, dto.SlowdownPercent)
		}
		if dto.Water && dto.Passable {
			return nil, fmt.Errorf("tile %d is passable water, but only impassable water is kept apart from the terrain", dto.Id)
		}
		if dto.Buildable && layer != data.TileLayer_Ground && layer != data.TileLayer_Foundation {
			return nil, fmt.Errorf("tile %d is buildable, but only ground and foundations are built on", dto.Id)
		}
		if dto.Water && layer != data.TileLayer_Ground {
			return nil, fmt.Errorf("tile %d is water, but only ground can be", dto.Id)
		}
		if layer == data.TileLayer_Ground && def.maxHealth > 0 {
			return nil, fmt.Errorf("tile %d is ground, which can't be damaged, since there'd be nothing left under it", dto.Id)
		}

		drops, err := dropTableFromDtos(dto.Drops, items)
		if err != nil {
			return nil, fmt.Errorf("tile %d has an invalid drop table: %w", dto.Id, err)
		}
		def.drops = drops

		tileDefs[data.TileEnum(dto.Id)] = def
	}

	return tileDefs, nil
}

var tileLayers = map[string]data.TileLayerEnum{
	"ground":     data.TileLayer_Ground,
	"foundation": data.TileLayer_Foundation,
	"cover":      data.TileLayer_Cover,
	"wall":       data.TileLayer_Wall,
}

// What a tile's layers add up to, kept for every tile of a chunk, since sight, light, paths and collisions ask for it
// tile by tile over large areas
type tileProps struct {
	passable bool
	// Impassable water, which stops walkers but not what flies over it
	water           bool
	opaque          bool
	buildable       bool
	harvestable     bool
	slowdownPercent uint8
}

// A foundation takes the ground's place in what the tile is like to walk on and build on, so that a bridge makes water
// passable. Walking speed is that of whichever of the cover, foundation and ground is on top. A tile on a layer that
// isn't in the atlas makes it opaque and impassable, like a tile outside the active chunks.
func (cc *ChunkContainer) combine(tiles data.TileLayers) tileProps {
	var defs [data.TileLayerCount]*tileDef
	for l, tile := range tiles {
		if tile == data.Tile_None {
			continue
		}
		def, ok := cc.atlas[tile]
		if !ok {
			return tileProps{opaque: true}
		}
		defs[l] = &def
	}

	base := defs[data.TileLayer_Ground]
	if base == nil {
		return tileProps{opaque: true}
	}
	if f := defs[data.TileLayer_Foundation]; f != nil {
		base = f
	}
	p := tileProps{
		passable:        base.passable,
		water:           base.water,
		buildable:       base.buildable,
		slowdownPercent: uint8(base.slowdownPercent),
	}
	if cover := defs[data.TileLayer_Cover]; cover != nil {
		p.slowdownPercent = uint8(cover.slowdownPercent)
	}
	for l, def := range defs {
		if def == nil {
			continue
		}
		p.opaque = p.opaque || def.opaque
		p.harvestable = p.harvestable || def.harvestable
		if data.TileLayerEnum(l) > data.TileLayer_Foundation {
			p.passable = p.passable && def.passable
		}
	}
	p.buildable = p.buildable && p.passable
	return p
}

func (p tileProps) collisionLayer() ecs.LayerMask {
	switch {
	case p.passable:
		return ecs.Layer_Floor
	case p.water:
		return ecs.Layer_Water
	default:
		return ecs.Layer_Wall
	}
}

// Tiles outside the active chunks count as opaque, since nothing about them is known to the simulation
func (cc *ChunkContainer) IsOpaqueTile(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return true
	}

	return c.propsAt(t).opaque || cc.opaqueFeaturesOf(c)[t]
}

func (cc *ChunkContainer) TileCollisionLayer(t utils.Vec2i) (ecs.LayerMask, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0, false
	}
	return c.propsAt(t).collisionLayer(), true
}

func (cc *ChunkContainer) IsBlockedAt(pos utils.Vec2f) bool {
	t := utils.WorldPosToTilePos(pos)
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return true
	}

	if !c.propsAt(t).passable {
		return true
	}

	b, walkThrough, ok := cc.featureCollider(c, t)
	return ok && !walkThrough && pos.X >= b[0].X && pos.X <= b[1].X && pos.Y >= b[0].Y && pos.Y <= b[1].Y
}

func (cc *ChunkContainer) HasTile(id data.TileEnum) bool {
	_, ok := cc.atlas[id]
	return ok
}
