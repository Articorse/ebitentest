package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
)

// The tiles the simulation runs on, so that systems and abilities can query and change them
type Terrain interface {
	IsOpaqueTile(t utils.Vec2i) bool
	// Loaded and without a roof
	IsOpenToSky(t utils.Vec2i) bool
	// Layer_Terrain for impassable tiles, Layer_Water for impassable water and Layer_Ground for passable tiles, whatever
	// stands on them; false if the tile isn't loaded
	TileLayer(t utils.Vec2i) (LayerMask, bool)
	// Inside an impassable tile, the collider of a feature that isn't walked through, or where nothing is loaded
	IsBlockedAt(pos utils.Vec2f) bool
	CanDamageTile(t utils.Vec2i) bool
	// False if the tile can't be damaged. What the tile drops, if this destroyed it.
	DamageTile(t utils.Vec2i, damage int) (damaged bool, drops *DropTable)
	// Whether there's room for a new tile: passable ground without a feature on it
	CanPlaceTile(t utils.Vec2i) bool
	// False if a tile can't be placed there, or the tile doesn't exist
	PlaceTile(t utils.Vec2i, tile data.TileEnum) bool
	// Features, such as trees, stand on a tile without replacing it, and are damaged separately from it
	CanDamageFeature(t utils.Vec2i) bool
	// False if there's no feature on the tile, or it can't be damaged. What the feature drops, if this destroyed it.
	DamageFeature(t utils.Vec2i, damage int) (damaged bool, drops *DropTable)
	// Whether there's room for a new feature: passable ground without a feature on it
	CanPlaceFeature(t utils.Vec2i) bool
	// False if a feature can't be placed there, or the feature doesn't exist
	PlaceFeature(t utils.Vec2i, feature data.FeatureEnum) bool
	// How fast walking over the tile is, in percent of walking on open ground, and 0 where it can't be walked on: an
	// impassable tile, one with a feature in the way, or one that isn't loaded
	WalkSpeedPercent(t utils.Vec2i) int
	// Calls lit for every feature giving light that reaches the square between min and max, with the light on its own
	// tile and how far away it fades to nothing
	LightSources(min, max utils.Vec2i, lit func(t utils.Vec2i, level uint8, rangeTiles int))
	// A blueprint is put down as a construction site, which is walked over and seen through. There's room for one on
	// ground that can be built on, without a feature or another site.
	CanPlaceSite(t utils.Vec2i) bool
	// False if a site can't be placed there, or the blueprint doesn't exist
	PlaceSite(t utils.Vec2i, blueprint data.BlueprintEnum) bool
	Site(t utils.Vec2i) (ConstructionSite, bool)
	// The tiles of the sites from first to last that are in the simulation, in tile order, row by row
	SitesIn(first, last utils.Vec2i) []utils.Vec2i
	// False unless it's still a blueprint, with nothing delivered
	RemoveSite(t utils.Vec2i) bool
	// How many of the items the site took, which is no more than it still needs of them
	DeliverToSite(t utils.Vec2i, item data.ItemEnum, count int) int
	// Works on a site that has everything delivered, and turns it into its tile once its health reaches the tile's,
	// unless it mustn't be finished yet, such as while something is in the tile's way. Worked is false if there's no such
	// site, or the work would finish one that mustn't be.
	BuildSite(t utils.Vec2i, work int, canFinish bool) (worked, built bool)
	// Where colonists look for work, which the player marks out and finished building spreads
	IsHome(t utils.Vec2i) bool
	// Marks the tiles from first to last as home, or not
	SetHome(first, last utils.Vec2i, home bool)
}

// Implemented by terrains that can answer for many tiles at once, and say when sight and light have to be worked out
// again, which light and vision use to avoid asking tile by tile and doing the same work every tick. Terrains without
// it, such as those of tests, are asked tile by tile and worked out every tick.
type SightTerrain interface {
	// Whether each tile of the w×h rectangle from from is opaque and open to the sky, row by row, as IsOpaqueTile and
	// IsOpenToSky would answer them. Either slice can be nil.
	TileFlags(from utils.Vec2i, w, h int, opaque, openToSky []bool)
	// Changes whenever the terrain changes in a way that could change what's seen or lit
	SightVersion() uint64
}

// Its health is how far building it has come, from 1 when it's placed up to that of the tile it builds
type ConstructionSite struct {
	Blueprint data.BlueprintEnum
	// For each of the blueprint's costs in turn, how many have been delivered
	Delivered []ItemCost
	Health    int
	MaxHealth int
}

// Only a plan until something's been delivered to it
func (x ConstructionSite) IsBlueprint() bool {
	for _, d := range x.Delivered {
		if d.Count > 0 {
			return false
		}
	}
	return true
}

// Every cost has been delivered, so building can start
func (x ConstructionSite) Supplied(def BlueprintDef) bool {
	for _, c := range def.costs {
		if x.Needs(c.Kind, def) > 0 {
			return false
		}
	}
	return true
}

func (x ConstructionSite) Needs(item data.ItemEnum, def BlueprintDef) int {
	needed := 0
	for _, c := range def.costs {
		if c.Kind == item {
			needed += c.Count
		}
	}
	for _, d := range x.Delivered {
		if d.Kind == item {
			needed -= d.Count
		}
	}
	return needed
}

type noTerrain struct{}

func (noTerrain) IsOpaqueTile(utils.Vec2i) bool           { return false }
func (noTerrain) IsOpenToSky(utils.Vec2i) bool            { return false }
func (noTerrain) TileLayer(utils.Vec2i) (LayerMask, bool) { return 0, false }
func (noTerrain) IsBlockedAt(utils.Vec2f) bool            { return false }
func (noTerrain) CanDamageTile(utils.Vec2i) bool          { return false }
func (noTerrain) DamageTile(utils.Vec2i, int) (bool, *DropTable) {
	return false, nil
}
func (noTerrain) CanPlaceTile(utils.Vec2i) bool             { return false }
func (noTerrain) PlaceTile(utils.Vec2i, data.TileEnum) bool { return false }
func (noTerrain) CanDamageFeature(utils.Vec2i) bool         { return false }
func (noTerrain) DamageFeature(utils.Vec2i, int) (bool, *DropTable) {
	return false, nil
}
func (noTerrain) CanPlaceFeature(utils.Vec2i) bool                { return false }
func (noTerrain) PlaceFeature(utils.Vec2i, data.FeatureEnum) bool { return false }
func (noTerrain) WalkSpeedPercent(utils.Vec2i) int                { return 100 }
func (noTerrain) LightSources(utils.Vec2i, utils.Vec2i, func(utils.Vec2i, uint8, int)) {
}
func (noTerrain) CanPlaceSite(utils.Vec2i) bool                         { return false }
func (noTerrain) PlaceSite(utils.Vec2i, data.BlueprintEnum) bool        { return false }
func (noTerrain) Site(utils.Vec2i) (ConstructionSite, bool)             { return ConstructionSite{}, false }
func (noTerrain) SitesIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i        { return nil }
func (noTerrain) RemoveSite(utils.Vec2i) bool                           { return false }
func (noTerrain) DeliverToSite(utils.Vec2i, data.ItemEnum, int) int     { return 0 }
func (noTerrain) IsHome(utils.Vec2i) bool                               { return false }
func (noTerrain) SetHome(utils.Vec2i, utils.Vec2i, bool)                {}
func (noTerrain) BuildSite(utils.Vec2i, int, bool) (worked, built bool) { return false, false }

func (x *ECSContainer) SetTerrain(terrain Terrain) {
	x.terrain = terrain
}

func (x *ECSContainer) Terrain() Terrain {
	return x.terrain
}

// Whether an entity is in the way of a tile being placed there: one whose physics collider reaches into it, or an
// item lying on it, which couldn't be picked up from inside a wall
func (x *ECSContainer) IsTileOccupied(t utils.Vec2i) bool {
	center := utils.TilePosToWorldPos(t)
	half := utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2}
	lo, hi := center.Subtract(half), center.Add(half)

	// Strictly inside, since something pressed flat against the tile, such as a player walking along a wall, isn't in
	// the way
	for e, col := range Query1(&x.PhysicsColliders, Without(&x.Disabled)) {
		aabb, err := x.ColliderWorldAABB(e, col)
		if err == nil && aabb[0].X < hi.X && aabb[1].X > lo.X && aabb[0].Y < hi.Y && aabb[1].Y > lo.Y {
			return true
		}
	}

	for e := range Query1(&x.Items, With(&x.Transforms), Without(&x.Disabled)) {
		if x.Parent(e) != -1 {
			continue
		}
		if pos, err := x.WorldPos(e); err == nil && utils.WorldPosToTilePos(pos) == t {
			return true
		}
	}

	return false
}

// How fast e walks on the tile under it, in percent. A tile that can't be walked through doesn't slow it, since
// something only stands partly on one, such as beside a tree's trunk.
func (x *ECSContainer) GroundSpeedPercent(e common.EntityId) int {
	pos, err := x.WorldPos(e)
	if err != nil {
		return 100
	}
	if speed := x.terrain.WalkSpeedPercent(utils.WorldPosToTilePos(pos)); speed > 0 {
		return speed
	}
	return 100
}
