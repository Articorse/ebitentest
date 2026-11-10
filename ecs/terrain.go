package ecs

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"math"
)

// The tiles the simulation runs on, so that systems and abilities can query and change them
type Terrain interface {
	IsOpaqueTile(t utils.Vec2i) bool
	// Loaded and without a roof
	IsOpenToSky(t utils.Vec2i) bool
	// Layer_Wall for impassable tiles, Layer_Water for impassable water and Layer_Floor for passable tiles, whatever
	// stands on them; false if the tile isn't loaded
	TileCollisionLayer(t utils.Vec2i) (LayerMask, bool)
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
	// False without a site there
	SetSiteOrder(t utils.Vec2i, order SiteOrder) bool
	SetSiteFaction(t utils.Vec2i, faction data.FactionEnum) bool
	Site(t utils.Vec2i) (ConstructionSite, bool)
	// The tiles of the sites from first to last that are in the simulation, in tile order, row by row
	SitesIn(first, last utils.Vec2i) []utils.Vec2i
	// False unless it's still a blueprint, with nothing delivered
	RemoveSite(t utils.Vec2i) bool
	// How much of the item the site took, which is no more than it still needs of them
	DeliverToSite(t utils.Vec2i, item data.ItemEnum, amount int) int
	// Works on a site that has everything delivered, and turns it into its tile once its health reaches the tile's,
	// unless it mustn't be finished yet, such as while something is in the tile's way. Worked is false if there's no such
	// site, or the work would finish one that mustn't be.
	BuildSite(t utils.Vec2i, work int, canFinish bool) (worked, built bool)
	// The tiles from first to last that are in the simulation with the feature on them, in tile order, row by row
	FeaturesIn(first, last utils.Vec2i, feature data.FeatureEnum) []utils.Vec2i
	// What lies on the tile
	GroundStack(t utils.Vec2i) (GroundStack, bool)
	// The tiles from first to last with something lying on them, in tile order, row by row
	GroundStacksIn(first, last utils.Vec2i) []utils.Vec2i
	// Puts the stack on the tile in place of what was there, or clears it with an empty one. False outside the active
	// chunks.
	SetGroundStack(t utils.Vec2i, s GroundStack) bool
	// Where colonists look for work and what work, which the player marks out, and finished building spreads home
	InArea(area data.AreaEnum, t utils.Vec2i) bool
	// The tiles from first to last in the area, in tile order, row by row
	AreaTiles(area data.AreaEnum, first, last utils.Vec2i) []utils.Vec2i
	// Puts the tiles from first to last in the area, or takes them out of it
	SetArea(area data.AreaEnum, first, last utils.Vec2i, in bool)
	// Whether the tile itself or its feature gives what's gathered on the harvest area
	IsHarvestable(t utils.Vec2i) bool
	// Whether something harvestable drops the kind
	IsHarvestItem(kind data.ItemEnum) bool
	// The tiles that something harvestable has been felled, broken or picked up from since this was last asked, in the
	// order it happened
	TakeHarvested() []utils.Vec2i
	// Which way the closed door on the tile runs and whose it is, false without one or while it's open
	ClosedDoor(t utils.Vec2i) (vertical bool, faction data.FactionEnum, ok bool)
	// Whether a body of the faction, reaching lo to hi tiles past the tile it's in the middle of, can walk from one tile
	// to touching the other, going through only its faction's doors. True where that can't be told, which is left to
	// paths to find out.
	Reachable(from, to utils.Vec2i, faction data.FactionEnum, lo, hi utils.Vec2i) bool
	// The tile closest to toward within radius tiles of it that such a body can walk to from the tile
	ClosestReachable(
		from, toward utils.Vec2i,
		faction data.FactionEnum,
		lo, hi utils.Vec2i,
		radius int,
	) (utils.Vec2i, bool)
	// The tiles a body of the faction reaching lo to hi round its own can walk to from the tile, as if the closed one
	// couldn't be walked on or reached over, up to limit of them. Enclosed if they're all there are, and none is next to
	// where the simulation ends, past which they may go on.
	Enclosure(
		from, closed utils.Vec2i,
		faction data.FactionEnum,
		lo, hi utils.Vec2i,
		limit int,
	) (tiles []utils.Vec2i, enclosed bool)
	// Whether there's a roof to build over the tile, on the roof area, or one to take down, on the no-roof area
	RoofJob(t utils.Vec2i) bool
	// The tiles from first to last with a roof job, in tile order, row by row
	RoofJobsIn(first, last utils.Vec2i) []utils.Vec2i
	// Works on the roof job over the tile, and done once it's built or taken down. Worked is false without a job there.
	WorkOnRoof(t utils.Vec2i, work int) (worked, done bool)
	// The air on the tile, per tile, in a room or outdoors, false on a wall or door or where nothing is active
	AirAt(t utils.Vec2i) (climate.Air, bool)
	// How warm the air on the tile is, in a room or outdoors
	AirTemperature(t utils.Vec2i) climate.Centicelsius
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
	// For each of the blueprint's costs in turn, how much has been delivered
	Delivered []ItemCost
	Health    int
	MaxHealth int
	Order     SiteOrder
	// Whose what's built will be
	Faction data.FactionEnum
}

// The sites placed together in one go make up an order, which is worked through from the first site placed to the
// last, and orders from the first placed to the last
type SiteOrder struct {
	// 0 for a site that isn't part of one, which is an order of its own
	Id uint64
	// Where it comes in its order, from 0
	Seq int
}

// Only a plan until something's been delivered to it
func (x ConstructionSite) IsBlueprint() bool {
	for _, d := range x.Delivered {
		if d.Amount > 0 {
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

// How far one go of the work builds a site up, which is never nothing
func SiteWorkStep(def BlueprintDef, maxHealth, work int) int {
	return max(1, maxHealth*work/def.Work())
}

func (x ConstructionSite) Needs(item data.ItemEnum, def BlueprintDef) int {
	needed := 0
	for _, c := range def.costs {
		if c.Kind == item {
			needed += c.Amount
		}
	}
	for _, d := range x.Delivered {
		if d.Kind == item {
			needed -= d.Amount
		}
	}
	return needed
}

type noTerrain struct{}

func (noTerrain) IsOpaqueTile(utils.Vec2i) bool                    { return false }
func (noTerrain) IsOpenToSky(utils.Vec2i) bool                     { return false }
func (noTerrain) TileCollisionLayer(utils.Vec2i) (LayerMask, bool) { return 0, false }
func (noTerrain) IsBlockedAt(utils.Vec2f) bool                     { return false }
func (noTerrain) CanDamageTile(utils.Vec2i) bool                   { return false }
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
func (noTerrain) SetSiteOrder(utils.Vec2i, SiteOrder) bool              { return false }
func (noTerrain) SetSiteFaction(utils.Vec2i, data.FactionEnum) bool     { return false }
func (noTerrain) Site(utils.Vec2i) (ConstructionSite, bool)             { return ConstructionSite{}, false }
func (noTerrain) SitesIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i        { return nil }
func (noTerrain) RemoveSite(utils.Vec2i) bool                           { return false }
func (noTerrain) DeliverToSite(utils.Vec2i, data.ItemEnum, int) int     { return 0 }
func (noTerrain) InArea(data.AreaEnum, utils.Vec2i) bool                { return false }
func (noTerrain) SetArea(data.AreaEnum, utils.Vec2i, utils.Vec2i, bool) {}
func (noTerrain) FeaturesIn(utils.Vec2i, utils.Vec2i, data.FeatureEnum) []utils.Vec2i {
	return nil
}
func (noTerrain) GroundStack(utils.Vec2i) (GroundStack, bool) {
	return GroundStack{}, false
}
func (noTerrain) GroundStacksIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i {
	return nil
}
func (noTerrain) SetGroundStack(utils.Vec2i, GroundStack) bool {
	return false
}
func (noTerrain) AreaTiles(data.AreaEnum, utils.Vec2i, utils.Vec2i) []utils.Vec2i {
	return nil
}
func (noTerrain) BuildSite(utils.Vec2i, int, bool) (worked, built bool) { return false, false }
func (noTerrain) IsHarvestable(utils.Vec2i) bool                        { return false }
func (noTerrain) IsHarvestItem(data.ItemEnum) bool                      { return false }
func (noTerrain) TakeHarvested() []utils.Vec2i                          { return nil }
func (noTerrain) ClosedDoor(utils.Vec2i) (bool, data.FactionEnum, bool) {
	return false, data.Faction_None, false
}
func (noTerrain) Reachable(utils.Vec2i, utils.Vec2i, data.FactionEnum, utils.Vec2i, utils.Vec2i) bool {
	return true
}
func (noTerrain) ClosestReachable(
	utils.Vec2i, utils.Vec2i, data.FactionEnum, utils.Vec2i, utils.Vec2i, int,
) (utils.Vec2i, bool) {
	return utils.Vec2i{}, false
}
func (noTerrain) Enclosure(utils.Vec2i, utils.Vec2i, data.FactionEnum, utils.Vec2i, utils.Vec2i, int) ([]utils.Vec2i, bool) {
	return nil, false
}
func (noTerrain) RoofJob(utils.Vec2i) bool                          { return false }
func (noTerrain) RoofJobsIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i { return nil }
func (noTerrain) AirAt(utils.Vec2i) (climate.Air, bool)             { return climate.Air{}, false }
func (noTerrain) AirTemperature(utils.Vec2i) climate.Centicelsius   { return 0 }
func (noTerrain) WorkOnRoof(utils.Vec2i, int) (worked, done bool)   { return false, false }

func (x *ECSContainer) SetTerrain(terrain Terrain) {
	x.terrain = terrain
}

func (x *ECSContainer) Terrain() Terrain {
	return x.terrain
}

// The tile what's at the position sees from and is seen on. A closed door runs through the middle of its tile, so on
// one it's the tile next to it on the position's side of the door.
func (x *ECSContainer) SightTile(pos utils.Vec2f) utils.Vec2i {
	t := utils.WorldPosToTilePos(pos)
	vertical, _, ok := x.terrain.ClosedDoor(t)
	if !ok {
		return t
	}
	center := utils.TilePosToWorldPos(t)
	switch {
	case vertical && pos.X < center.X:
		t.X--
	case vertical:
		t.X++
	case pos.Y < center.Y:
		t.Y--
	default:
		t.Y++
	}
	return t
}

// As WalkSpeedPercent, but 0 on a closed door that won't open for the entity, which is any that isn't of its faction.
// One without a faction is no one's. A body bigger than a tile walks on a tile only where every tile round it that it
// takes up standing in its middle can be walked on, at the speed of the tile it's in the middle of, so that its paths
// and the straight lines it walks keep it clear of walls.
func (x *ECSContainer) WalkSpeedOf(e common.EntityId) func(utils.Vec2i) int {
	faction := x.FactionOf(e)
	tile := func(t utils.Vec2i) int {
		if _, door, ok := x.terrain.ClosedDoor(t); ok && (door == data.Faction_None || door != faction) {
			return 0
		}
		return x.terrain.WalkSpeedPercent(t)
	}
	lo, hi := x.bodyReach(e)
	if lo == (utils.Vec2i{}) && hi == (utils.Vec2i{}) {
		return tile
	}
	everyTile := func(t utils.Vec2i) int {
		for dy := lo.Y; dy <= hi.Y; dy++ {
			for dx := lo.X; dx <= hi.X; dx++ {
				if tile(utils.Vec2i{X: t.X + dx, Y: t.Y + dy}) <= 0 {
					return 0
				}
			}
		}
		return tile(t)
	}
	bodies, ok := x.terrain.(BodyTerrain)
	if !ok {
		return everyTile
	}
	speeds := bodies.BodySpeeds(lo, hi)
	return func(t utils.Vec2i) int {
		speed, door := speeds(t)
		if door && speed > 0 {
			return everyTile(t)
		}
		return speed
	}
}

// Terrain that keeps where bodies bigger than a tile fit, rather than their every tile being looked at each time
type BodyTerrain interface {
	// How fast a body reaching lo to hi tiles round a tile walks in the middle of it, 0 where a tile it would take up
	// can't be walked on, and whether one of them is a door, whose faction the terrain leaves to the walker
	BodySpeeds(lo, hi utils.Vec2i) func(t utils.Vec2i) (speed int, door bool)
}

// Whether the entity can walk from one tile to touching the other, as Terrain.Reachable says for its body and faction
func (x *ECSContainer) Reachable(e common.EntityId, from, to utils.Vec2i) bool {
	lo, hi := x.bodyReach(e)
	return x.terrain.Reachable(from, to, x.FactionOf(e), lo, hi)
}

// The tiles the entity can walk to from the tile with the closed one in its way, as Terrain.Enclosure says for its body
// and faction
func (x *ECSContainer) Enclosure(e common.EntityId, from, closed utils.Vec2i, limit int) ([]utils.Vec2i, bool) {
	lo, hi := x.bodyReach(e)
	return x.terrain.Enclosure(from, closed, x.FactionOf(e), lo, hi, limit)
}

// The tile closest to toward, within data.PathClosestTiles of it, that the entity can walk to from the tile
func (x *ECSContainer) ClosestReachable(e common.EntityId, from, toward utils.Vec2i) (utils.Vec2i, bool) {
	lo, hi := x.bodyReach(e)
	return x.terrain.ClosestReachable(from, toward, x.FactionOf(e), lo, hi, data.PathClosestTiles)
}

// How many tiles the entity's physics collider reaches past the tile it's in the middle of, up and left as negative
// numbers and down and right as positive. Strictly, so that a body reaching exactly to the next tile's edge stays out of
// it.
func (x *ECSContainer) bodyReach(e common.EntityId) (lo, hi utils.Vec2i) {
	col, ok := x.PhysicsColliders.Get(e)
	if !ok {
		return lo, hi
	}
	local, err := x.transformedShapes(e, col)
	if err != nil {
		return lo, hi
	}
	aabb := shapes.BoundingAABB(local)
	past := func(d float64) int {
		return max(0, int(math.Ceil((d-data.TileSize/2)/data.TileSize)))
	}
	return utils.Vec2i{X: -past(-aabb[0].X), Y: -past(-aabb[0].Y)}, utils.Vec2i{X: past(aabb[1].X), Y: past(aabb[1].Y)}
}

// Whether an entity is in the way of a tile being placed there: one whose physics collider reaches into it, or an
// item lying on it, which couldn't be picked up from inside a wall
func (x *ECSContainer) IsTileOccupied(t utils.Vec2i) bool {
	_, onGround := x.terrain.GroundStack(t)
	return onGround || x.IsEntityInTile(t)
}

// Whether something's physics collider reaches into the tile
func (x *ECSContainer) IsEntityInTile(t utils.Vec2i) bool {
	for e, col := range Query1(&x.PhysicsColliders, Without(&x.Disabled)) {
		if x.colliderOverlapsTile(e, col, t) {
			return true
		}
	}
	return false
}

// Whether e's physics collider is in the way of the tile
func (x *ECSContainer) OverlapsTile(e common.EntityId, t utils.Vec2i) bool {
	col, ok := x.PhysicsColliders.Get(e)
	return ok && x.colliderOverlapsTile(e, col, t)
}

// By the collider's shapes rather than the box around them, which reaches into tiles a round or slanted body doesn't
func (x *ECSContainer) colliderOverlapsTile(e common.EntityId, col *PhysicsCollider, t utils.Vec2i) bool {
	pos, err := x.WorldPos(e)
	if err != nil {
		return false
	}
	local, err := x.transformedShapes(e, col)
	return err == nil && shapesOverlapTile(local, pos, t)
}

// Strictly inside, since something pressed flat against the tile, such as a player walking along a wall, isn't in the
// way
func shapesOverlapTile(local []shapes.Shape, pos utils.Vec2f, t utils.Vec2i) bool {
	center := utils.TilePosToWorldPos(t).Subtract(pos)
	half := utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2}
	for _, s := range local {
		if shapes.OverlapsBox(s, center.Subtract(half), center.Add(half)) {
			return true
		}
	}
	return false
}

func (x *ECSContainer) transformedShapes(e common.EntityId, col Collider) ([]shapes.Shape, error) {
	local := make([]shapes.Shape, len(col.Shapes()))
	for i, s := range col.Shapes() {
		var err error
		if local[i], err = x.TransformShape(e, s); err != nil {
			return nil, err
		}
	}
	return local, nil
}

// The tiles shapes at the position reach into, row by row, which is what a body standing there takes up
func ShapesFootprint(local []shapes.Shape, pos utils.Vec2f) []utils.Vec2i {
	aabb := shapes.BoundingAABB(local)
	first := utils.WorldPosToTilePos(aabb[0].Add(pos))
	last := utils.WorldPosToTilePos(aabb[1].Add(pos))
	tiles := []utils.Vec2i{}
	for y := first.Y; y <= last.Y; y++ {
		for x := first.X; x <= last.X; x++ {
			if t := (utils.Vec2i{X: x, Y: y}); shapesOverlapTile(local, pos, t) {
				tiles = append(tiles, t)
			}
		}
	}
	return tiles
}

// The tiles the entity's physics collider takes up, none without one
func (x *ECSContainer) Footprint(e common.EntityId) []utils.Vec2i {
	col, ok := x.PhysicsColliders.Get(e)
	if !ok {
		return nil
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return nil
	}
	local, err := x.transformedShapes(e, col)
	if err != nil {
		return nil
	}
	return ShapesFootprint(local, pos)
}

// Whether a body of the shapes fits at the position: every tile it would take up walkable, and none with something in
// it already
func (x *ECSContainer) FitsAt(local []shapes.Shape, pos utils.Vec2f) bool {
	for _, t := range ShapesFootprint(local, pos) {
		if x.terrain.WalkSpeedPercent(t) <= 0 || x.IsEntityInTile(t) {
			return false
		}
	}
	return true
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
