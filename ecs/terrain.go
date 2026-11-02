package ecs

import "ebittest/utils"

// The tiles the simulation runs on, so that systems and abilities can query and change them
type Terrain interface {
	IsOpaqueTile(t utils.Vec2i) bool
	// Layer_Terrain for impassable tiles and Layer_Ground for passable ones; false if the tile isn't loaded
	TileLayer(t utils.Vec2i) (LayerMask, bool)
	// False if the tile can't be damaged
	DamageTile(t utils.Vec2i, damage int) bool
}

type noTerrain struct{}

func (noTerrain) IsOpaqueTile(utils.Vec2i) bool           { return false }
func (noTerrain) TileLayer(utils.Vec2i) (LayerMask, bool) { return 0, false }
func (noTerrain) DamageTile(utils.Vec2i, int) bool        { return false }

func (x *ECSContainer) SetTerrain(terrain Terrain) {
	x.terrain = terrain
}

func (x *ECSContainer) Terrain() Terrain {
	return x.terrain
}
