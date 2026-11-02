package ecs

import "ebittest/utils"

// The tiles the simulation runs on, so that systems and abilities can query and change them
type Terrain interface {
	IsOpaqueTile(t utils.Vec2i) bool
	// Layer_Terrain for impassable tiles and Layer_Ground for passable ones; false if the tile isn't loaded
	TileLayer(t utils.Vec2i) (LayerMask, bool)
	CanDamageTile(t utils.Vec2i) bool
	// False if the tile can't be damaged
	DamageTile(t utils.Vec2i, damage int) bool
	// Features, such as trees, stand on a tile without replacing it, and are damaged separately from it
	CanDamageFeature(t utils.Vec2i) bool
	// False if there's no feature on the tile, or it can't be damaged
	DamageFeature(t utils.Vec2i, damage int) bool
}

type noTerrain struct{}

func (noTerrain) IsOpaqueTile(utils.Vec2i) bool           { return false }
func (noTerrain) TileLayer(utils.Vec2i) (LayerMask, bool) { return 0, false }
func (noTerrain) CanDamageTile(utils.Vec2i) bool          { return false }
func (noTerrain) DamageTile(utils.Vec2i, int) bool        { return false }
func (noTerrain) CanDamageFeature(utils.Vec2i) bool       { return false }
func (noTerrain) DamageFeature(utils.Vec2i, int) bool     { return false }

func (x *ECSContainer) SetTerrain(terrain Terrain) {
	x.terrain = terrain
}

func (x *ECSContainer) Terrain() Terrain {
	return x.terrain
}
