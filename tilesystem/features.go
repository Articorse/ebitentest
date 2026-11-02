package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

type featureDef struct {
	imageAsset common.ImageAssetTag
	// Where the image's center is, relative to the center of the feature's tile
	imageOffset utils.Vec2f
	spriteLayer uint8

	hasCollider bool
	// World-space bounds relative to the center of the feature's tile
	colliderMin, colliderMax utils.Vec2f

	// Not positive for features that can't be damaged
	maxHealth     int
	destroyedInto data.FeatureEnum
	// Nil for features that drop nothing when destroyed
	drops *ecs.DropTable
}

type featureDefDTO struct {
	Id          uint16              `json:"id"`
	ImageAsset  string              `json:"imageAsset"`
	ImageOffset utils.Vec2f         `json:"imageOffset"`
	SpriteLayer uint8               `json:"spriteLayer"`
	Collider    *featureColliderDTO `json:"collider"`
	MaxHealth   int                 `json:"maxHealth"`
	// Required for features that can be damaged
	DestroyedInto *uint16          `json:"destroyedInto"`
	Drops         []dropOutcomeDTO `json:"drops"`
}

type dropOutcomeDTO struct {
	ChancePercent float64       `json:"chance"`
	Items         []itemDropDTO `json:"items"`
}

type itemDropDTO struct {
	Kind  uint16 `json:"kind"`
	Count int    `json:"count"`
}

type featureColliderDTO struct {
	Width  float64     `json:"width"`
	Height float64     `json:"height"`
	Offset utils.Vec2f `json:"offset"`
}

func (cc *ChunkContainer) GenerateFeatureAtlasFromJson(input string, items *ecs.ItemAtlas) error {
	f, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("error reading feature atlas json file: %w", err)
	}

	var dtos []featureDefDTO
	if err := json.Unmarshal(f, &dtos); err != nil {
		return fmt.Errorf("error unmarshalling feature atlas json: %w", err)
	}

	defs := make(map[data.FeatureEnum]featureDef, len(dtos))
	for _, dto := range dtos {
		def := featureDef{
			imageAsset:  common.ImageAssetTag(dto.ImageAsset),
			imageOffset: dto.ImageOffset,
			spriteLayer: dto.SpriteLayer,
		}

		if c := dto.Collider; c != nil {
			if c.Width <= 0 || c.Height <= 0 {
				return fmt.Errorf("feature %d has a collider of %v×%v, which isn't positive", dto.Id, c.Width, c.Height)
			}
			def.hasCollider = true
			def.colliderMin = c.Offset.Subtract(utils.Vec2f{X: c.Width / 2, Y: c.Height / 2})
			def.colliderMax = c.Offset.Add(utils.Vec2f{X: c.Width / 2, Y: c.Height / 2})
		}

		def.maxHealth = dto.MaxHealth
		if def.maxHealth > 0 {
			if dto.DestroyedInto == nil {
				return fmt.Errorf("feature %d can be damaged, but doesn't say what it's destroyed into", dto.Id)
			}
			def.destroyedInto = data.FeatureEnum(*dto.DestroyedInto)
		}

		drops, err := dropTableFromDtos(dto.Drops, items)
		if err != nil {
			return fmt.Errorf("feature %d has an invalid drop table: %w", dto.Id, err)
		}
		def.drops = drops

		defs[data.FeatureEnum(dto.Id)] = def
	}

	for id, def := range defs {
		if _, ok := defs[def.destroyedInto]; def.maxHealth > 0 && !ok {
			return fmt.Errorf("feature %d is destroyed into feature %d, which isn't defined", id, def.destroyedInto)
		}
	}

	cc.featureAtlas = defs
	return nil
}

func (cc *ChunkContainer) CanDamageFeature(t utils.Vec2i) bool {
	_, ok := cc.FeatureHealth(t)
	return ok
}

func (cc *ChunkContainer) FeatureHealth(t utils.Vec2i) (health int, ok bool) {
	c, def, ok := cc.featureOn(t)
	if !ok || def.maxHealth <= 0 {
		return 0, false
	}

	if h, ok := c.featureHealth[t]; ok {
		return h, true
	}
	return def.maxHealth, true
}

// Returns false for features that can't be damaged, or tiles without one in an active chunk. Returns what the feature
// drops, if this destroyed it.
func (cc *ChunkContainer) DamageFeature(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	c, def, ok := cc.featureOn(t)
	if !ok || def.maxHealth <= 0 {
		return false, nil
	}

	health := def.maxHealth
	if h, ok := c.featureHealth[t]; ok {
		health = h
	}
	health -= damage
	c.terrainModified = true

	if health > 0 {
		c.featureHealth[t] = health
		return true, nil
	}

	delete(c.featureHealth, t)
	c.features[t] = def.destroyedInto
	return true, def.drops
}

func (cc *ChunkContainer) featureOn(t utils.Vec2i) (*chunk, featureDef, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return nil, featureDef{}, false
	}

	f, ok := c.features[t]
	if !ok {
		return nil, featureDef{}, false
	}

	def, ok := cc.featureAtlas[f]
	return c, def, ok
}

func (cc *ChunkContainer) FeatureAt(t utils.Vec2i) (data.FeatureEnum, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return data.Feature_None, false
	}

	f, ok := c.features[t]
	return f, ok
}

// The world-space bounds of the collider of the feature on the tile, if it has one
func (cc *ChunkContainer) featureCollider(c *chunk, t utils.Vec2i) ([2]utils.Vec2f, bool) {
	f, ok := c.features[t]
	if !ok {
		return [2]utils.Vec2f{}, false
	}

	def, ok := cc.featureAtlas[f]
	if !ok || !def.hasCollider {
		return [2]utils.Vec2f{}, false
	}

	center := utils.TilePosToWorldPos(t)
	return [2]utils.Vec2f{center.Add(def.colliderMin), center.Add(def.colliderMax)}, true
}

func (cc *ChunkContainer) featureSprite(t utils.Vec2i, f data.FeatureEnum) (common.StaticSprite, bool) {
	def, ok := cc.featureAtlas[f]
	if !ok {
		return common.StaticSprite{}, false
	}

	return common.StaticSprite{
		ImageAsset: def.imageAsset,
		Pos:        utils.TilePosToWorldPos(t).Add(def.imageOffset),
		Layer:      def.spriteLayer,
	}, true
}

// The sprites of the features in the active chunks whose tiles are within the bounds, in tile order so that drawing
// them is the same every frame
func (cc *ChunkContainer) FeatureSprites(bounds [2]utils.Vec2f) []common.StaticSprite {
	minTile, maxTile := utils.WorldPosToTilePos(bounds[0]), utils.WorldPosToTilePos(bounds[1])
	minChunk := utils.Vec2i{X: floorDiv(minTile.X, data.ChunkSize), Y: floorDiv(minTile.Y, data.ChunkSize)}
	maxChunk := utils.Vec2i{X: floorDiv(maxTile.X, data.ChunkSize), Y: floorDiv(maxTile.Y, data.ChunkSize)}

	type placed struct {
		tile   utils.Vec2i
		sprite common.StaticSprite
	}
	found := []placed{}

	for cy := minChunk.Y; cy <= maxChunk.Y; cy++ {
		for cx := minChunk.X; cx <= maxChunk.X; cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok {
				continue
			}

			for t, f := range c.features {
				if t.X < minTile.X || t.X > maxTile.X || t.Y < minTile.Y || t.Y > maxTile.Y {
					continue
				}
				if sprite, ok := cc.featureSprite(t, f); ok {
					found = append(found, placed{t, sprite})
				}
			}
		}
	}

	slices.SortFunc(found, func(a, b placed) int {
		return cmp.Or(cmp.Compare(a.tile.Y, b.tile.Y), cmp.Compare(a.tile.X, b.tile.X))
	})

	sprites := make([]common.StaticSprite, len(found))
	for i, p := range found {
		sprites[i] = p.sprite
	}
	return sprites
}

// Nil for no outcomes. Every item is checked against the item atlas.
func dropTableFromDtos(dtos []dropOutcomeDTO, items *ecs.ItemAtlas) (*ecs.DropTable, error) {
	if len(dtos) == 0 {
		return nil, nil
	}

	outcomes := make([]ecs.DropOutcome, len(dtos))
	for i, o := range dtos {
		outcomes[i].ChancePercent = o.ChancePercent
		for _, item := range o.Items {
			if _, ok := items.Def(data.ItemEnum(item.Kind)); !ok {
				return nil, fmt.Errorf("item %d isn't in the item atlas", item.Kind)
			}
			outcomes[i].Items = append(outcomes[i].Items, ecs.ItemDrop{Kind: data.ItemEnum(item.Kind), Count: item.Count})
		}
	}
	return ecs.NewDropTable(outcomes...)
}
