package ecs

import (
	"ebittest/data"
	"ebittest/utils"
)

// The tiles the simulation runs on, so that systems and abilities can query and change them
type Terrain interface {
	IsOpaqueTile(t utils.Vec2i) bool
	// Loaded and without a roof
	IsOpenToSky(t utils.Vec2i) bool
	// Layer_Terrain for impassable tiles and Layer_Ground for passable ones, whatever stands on them; false if the tile
	// isn't loaded
	TileLayer(t utils.Vec2i) (LayerMask, bool)
	// Inside an impassable tile, a feature's collider, or where nothing is loaded
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
