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
	// Not positive for tiles that can't be damaged
	maxHealth     int
	destroyedInto data.TileEnum
	passable      bool
	// An impassable tile that's water is on Layer_Water rather than Layer_Terrain, so that it stops walkers but not
	// what flies over it
	water  bool
	opaque bool
	// How much slower walking over it is, which only paths take into account for now
	slowdownPercent int
	// Nil for tiles that drop nothing when destroyed
	drops *ecs.DropTable
}

type promotedTile struct {
	currentHealth int
}

type tileDefDTO struct {
	Id         uint16 `json:"id"`
	ImageAsset string `json:"imageAsset"`
	MaxHealth  int    `json:"maxHealth"`
	// Required for tiles that can be damaged
	DestroyedInto   *uint16          `json:"destroyedInto"`
	Passable        bool             `json:"passable"`
	Water           bool             `json:"water"`
	Opaque          bool             `json:"opaque"`
	SlowdownPercent int              `json:"slowdownPercent"`
	Drops           []dropOutcomeDTO `json:"drops"`
}

func dtosToDefsMap(dtos []tileDefDTO, items *ecs.ItemAtlas, assets *assetmanager.AssetManager) (map[data.TileEnum]tileDef, error) {
	tileDefs := make(map[data.TileEnum]tileDef, len(dtos))
	for _, dto := range dtos {
		asset, err := assets.GetAsset(common.ImageAssetTag(dto.ImageAsset))
		if err != nil {
			return nil, fmt.Errorf("tile %d's image: %w", dto.Id, err)
		}
		img := asset.Frames[0]

		def := tileDef{
			image:     img,
			maxHealth: dto.MaxHealth,
			passable:  dto.Passable,
			water:     dto.Water,
			opaque:    dto.Opaque,

			slowdownPercent: dto.SlowdownPercent,
		}
		if dto.SlowdownPercent < 0 || dto.SlowdownPercent >= 100 {
			return nil, fmt.Errorf("tile %d slows walking by %d%%, want 0 to 99, or the tile impassable", dto.Id, dto.SlowdownPercent)
		}
		if dto.Water && dto.Passable {
			return nil, fmt.Errorf("tile %d is passable water, but only impassable water is kept apart from the terrain", dto.Id)
		}

		if def.maxHealth > 0 {
			if dto.DestroyedInto == nil {
				return nil, fmt.Errorf("tile %d can be damaged, but doesn't say what it's destroyed into", dto.Id)
			}
			def.destroyedInto = data.TileEnum(*dto.DestroyedInto)
		}

		drops, err := dropTableFromDtos(dto.Drops, items)
		if err != nil {
			return nil, fmt.Errorf("tile %d has an invalid drop table: %w", dto.Id, err)
		}
		def.drops = drops

		tileDefs[data.TileEnum(dto.Id)] = def
	}

	for id, def := range tileDefs {
		if _, ok := tileDefs[def.destroyedInto]; def.maxHealth > 0 && !ok {
			return nil, fmt.Errorf("tile %d is destroyed into tile %d, which isn't defined", id, def.destroyedInto)
		}
	}

	return tileDefs, nil
}

// Tiles outside the active chunks count as opaque, since nothing about them is known to the simulation
func (cc *ChunkContainer) IsOpaqueTile(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return true
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	return !ok || def.opaque
}

func (cc *ChunkContainer) TileLayer(t utils.Vec2i) (ecs.LayerMask, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0, false
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	if !ok {
		return 0, false
	}

	return def.layer(), true
}

func (def tileDef) layer() ecs.LayerMask {
	switch {
	case def.passable:
		return ecs.Layer_Ground
	case def.water:
		return ecs.Layer_Water
	default:
		return ecs.Layer_Terrain
	}
}

func (cc *ChunkContainer) IsBlockedAt(pos utils.Vec2f) bool {
	t := utils.WorldPosToTilePos(pos)
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return true
	}

	if def, ok := cc.atlas[c.getTileDefId(t)]; !ok || !def.passable {
		return true
	}

	b, walkThrough, ok := cc.featureCollider(c, t)
	return ok && !walkThrough && pos.X >= b[0].X && pos.X <= b[1].X && pos.Y >= b[0].Y && pos.Y <= b[1].Y
}

func (cc *ChunkContainer) HasTile(id data.TileEnum) bool {
	_, ok := cc.atlas[id]
	return ok
}
