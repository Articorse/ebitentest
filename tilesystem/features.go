package tilesystem

import (
	"cmp"
	"ebittest/assetmanager"
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
	// What walks through the collider still hits the feature with what it carries
	walkThrough bool

	// Nil for a still image
	animation []featureFrame
	// The light on the feature's own tile, fading to nothing lightRangeTiles away. 0 for features that don't give light.
	light           uint8
	lightRangeTiles int

	// Not positive for features that can't be damaged
	maxHealth     int
	destroyedInto data.FeatureEnum
	// Nil for features that drop nothing when destroyed
	drops *ecs.DropTable
	// Gives what's gathered on the harvest area, which keeps the area there for as long as it's left
	harvestable bool
	// Blocks sight and light, as an opaque tile does
	opaque bool
	// Nil for anything but a door
	door *doorDef
}

// A door is one of four features of its kind, one for each way it can face, open and closed, which it's swapped
// between
type doorDef struct {
	kind     string
	vertical bool
	open     bool
}

type featureDefDTO struct {
	Id          uint16              `json:"id"`
	ImageAsset  string              `json:"imageAsset"`
	ImageOffset utils.Vec2f         `json:"imageOffset"`
	SpriteLayer uint8               `json:"spriteLayer"`
	Animation   []featureFrame      `json:"animation"`
	Collider    *featureColliderDTO `json:"collider"`
	Light       *featureLightDTO    `json:"light"`
	MaxHealth   int                 `json:"maxHealth"`
	// Required for features that can be damaged, and 0 for those that leave nothing behind
	DestroyedInto *uint16          `json:"destroyedInto"`
	Drops         []dropOutcomeDTO `json:"drops"`
	Harvestable   bool             `json:"harvestable"`
	Opaque        bool             `json:"opaque"`
	Door          *featureDoorDTO  `json:"door"`
}

type featureDoorDTO struct {
	Kind     string `json:"kind"`
	Vertical bool   `json:"vertical"`
	Open     bool   `json:"open"`
}

type featureFrame struct {
	FrameIdx   int `json:"frame"`
	DurationMs int `json:"durationMs"`
}

type featureLightDTO struct {
	Level      uint8 `json:"level"`
	RangeTiles int   `json:"rangeTiles"`
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
	Width       float64     `json:"width"`
	Height      float64     `json:"height"`
	Offset      utils.Vec2f `json:"offset"`
	WalkThrough bool        `json:"walkThrough"`
}

// Images, and the frames animations show, are checked against the asset list
func (cc *ChunkContainer) GenerateFeatureAtlasFromJson(input string, items *ecs.ItemAtlas, assets *assetmanager.AssetManager) error {
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
		asset, err := assets.GetAsset(def.imageAsset)
		if err != nil {
			return fmt.Errorf("feature %d's image: %w", dto.Id, err)
		}

		if c := dto.Collider; c != nil {
			if c.Width <= 0 || c.Height <= 0 {
				return fmt.Errorf("feature %d has a collider of %v×%v, which isn't positive", dto.Id, c.Width, c.Height)
			}
			def.hasCollider = true
			def.colliderMin = c.Offset.Subtract(utils.Vec2f{X: c.Width / 2, Y: c.Height / 2})
			def.colliderMax = c.Offset.Add(utils.Vec2f{X: c.Width / 2, Y: c.Height / 2})
			def.walkThrough = c.WalkThrough
		}

		for _, frame := range dto.Animation {
			if frame.FrameIdx < 0 || frame.FrameIdx >= len(asset.Frames) || frame.DurationMs <= 0 {
				return fmt.Errorf("feature %d has an animation frame %d of %d lasting %dms", dto.Id, frame.FrameIdx, len(asset.Frames), frame.DurationMs)
			}
		}
		def.animation = dto.Animation

		if l := dto.Light; l != nil {
			if l.Level == 0 || l.RangeTiles <= 0 {
				return fmt.Errorf("feature %d gives light %d up to %d tiles away, which isn't positive", dto.Id, l.Level, l.RangeTiles)
			}
			def.light, def.lightRangeTiles = l.Level, l.RangeTiles
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
		def.harvestable = dto.Harvestable
		def.opaque = dto.Opaque
		if d := dto.Door; d != nil {
			def.door = &doorDef{kind: d.Kind, vertical: d.Vertical, open: d.Open}
		}

		defs[data.FeatureEnum(dto.Id)] = def
	}

	maxLightRange := 0
	for id, def := range defs {
		maxLightRange = max(maxLightRange, def.lightRangeTiles)
		if _, ok := defs[def.destroyedInto]; def.maxHealth > 0 && def.destroyedInto != data.Feature_None && !ok {
			return fmt.Errorf("feature %d is destroyed into feature %d, which isn't defined", id, def.destroyedInto)
		}
		if def.door != nil {
			for _, vertical := range []bool{false, true} {
				for _, open := range []bool{false, true} {
					other, ok := doorVariant(defs, def.door.kind, vertical, open)
					if !ok || defs[other].maxHealth != def.maxHealth {
						return fmt.Errorf("door %d of kind %q has no %s door with its health, open %v", id, def.door.kind, map[bool]string{false: "horizontal", true: "vertical"}[vertical], open)
					}
				}
			}
		}
	}

	cc.featureAtlas = defs
	cc.maxLightRange = maxLightRange
	return nil
}

func (cc *ChunkContainer) HasFeature(id data.FeatureEnum) bool {
	_, ok := cc.featureAtlas[id]
	return ok
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
	cc.setFeature(c, t, def.destroyedInto)
	return true, def.drops
}

// Whether there's room for a new feature: passable ground without a feature on it
// Not on storage, which is kept clear for what's put away there
func (cc *ChunkContainer) CanPlaceFeature(t utils.Vec2i) bool {
	return cc.CanPlaceTile(t) && !cc.InArea(data.Area_Storage, t)
}

// Returns false where a feature can't be placed, or the feature isn't in the atlas
func (cc *ChunkContainer) PlaceFeature(t utils.Vec2i, f data.FeatureEnum) bool {
	if _, ok := cc.featureAtlas[f]; !ok || !cc.CanPlaceFeature(t) {
		return false
	}

	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	cc.setFeature(c, t, f)
	return true
}

func (cc *ChunkContainer) setFeature(c *chunk, t utils.Vec2i, f data.FeatureEnum) {
	if cc.featureAtlas[c.features[t]].harvestable {
		cc.noteHarvested(t)
	}
	if f == data.Feature_None {
		delete(c.features, t)
	} else {
		c.features[t] = f
	}
	c.terrainModified = true
	c.lightsIndexed = false
	c.walkSpeeds = nil
	c.opaqueFeatures = nil
	cc.sightVersion++
}

// The tiles with a feature that blocks sight, found when they're first asked for after the features change, since
// sight asks tile by tile over large squares
func (cc *ChunkContainer) opaqueFeaturesOf(c *chunk) map[utils.Vec2i]bool {
	if c.opaqueFeatures == nil {
		c.opaqueFeatures = map[utils.Vec2i]bool{}
		for t, f := range c.features {
			if cc.featureAtlas[f].opaque {
				c.opaqueFeatures[t] = true
			}
		}
	}
	return c.opaqueFeatures
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

func (cc *ChunkContainer) FeaturesIn(first, last utils.Vec2i, feature data.FeatureEnum) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for cy := floorDiv(first.Y, data.ChunkSize); cy <= floorDiv(last.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(first.X, data.ChunkSize); cx <= floorDiv(last.X, data.ChunkSize); cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok {
				continue
			}
			for t, f := range c.features {
				if f == feature && t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y {
					tiles = append(tiles, t)
				}
			}
		}
	}
	// Map order differs from run to run, and what colonists choose mustn't
	slices.SortFunc(tiles, func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return tiles
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
func (cc *ChunkContainer) featureCollider(c *chunk, t utils.Vec2i) (bounds [2]utils.Vec2f, walkThrough bool, ok bool) {
	f, ok := c.features[t]
	if !ok {
		return [2]utils.Vec2f{}, false, false
	}

	def, ok := cc.featureAtlas[f]
	if !ok || !def.hasCollider {
		return [2]utils.Vec2f{}, false, false
	}

	center := utils.TilePosToWorldPos(t)
	return [2]utils.Vec2f{center.Add(def.colliderMin), center.Add(def.colliderMax)}, def.walkThrough, true
}

// The animation's frame at the time, or the first frame for features without an animation
func (cc *ChunkContainer) featureSprite(t utils.Vec2i, f data.FeatureEnum, ms uint64) (common.StaticSprite, bool) {
	def, ok := cc.featureAtlas[f]
	if !ok {
		return common.StaticSprite{}, false
	}

	return common.StaticSprite{
		ImageAsset: def.imageAsset,
		Frame:      animationFrame(def.animation, ms),
		Pos:        utils.TilePosToWorldPos(t).Add(def.imageOffset),
		Layer:      def.spriteLayer,
	}, true
}

func animationFrame(frames []featureFrame, ms uint64) int {
	total := 0
	for _, f := range frames {
		total += f.DurationMs
	}
	if total == 0 {
		return 0
	}

	at := int(ms % uint64(total))
	for _, f := range frames {
		if at < f.DurationMs {
			return f.FrameIdx
		}
		at -= f.DurationMs
	}
	return 0
}

// The sprites of the features and construction sites in the active chunks whose tiles are within the bounds, features at
// the time in their animations, in tile order so that drawing them is the same every frame
func (cc *ChunkContainer) FeatureSprites(bounds [2]utils.Vec2f, ms uint64) []common.StaticSprite {
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
				if sprite, ok := cc.featureSprite(t, f, ms); ok {
					found = append(found, placed{t, sprite})
				}
			}
			for t, s := range c.sites {
				if t.X < minTile.X || t.X > maxTile.X || t.Y < minTile.Y || t.Y > maxTile.Y || s.isBlueprint() {
					continue
				}
				if sprite, ok := cc.siteSprite(t, s); ok {
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
