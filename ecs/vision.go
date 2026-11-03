package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
)

// Sees the tiles in line of sight that are lit enough for how far away they are
type Vision struct {
	// How far it sees in full light with nothing impairing it
	baseRangeTiles int

	// What was visible at the last update, as a square around origin. It's derived from the world, so it isn't saved.
	origin     utils.Vec2i
	rangeTiles int
	visible    []bool
	light      LightMap
}

func NewVisionComponent(baseRangeTiles int) *Vision {
	return &Vision{baseRangeTiles: baseRangeTiles}
}

func (x *Vision) BaseRangeTiles() int {
	return x.baseRangeTiles
}

// A tile is seen if it's in line of sight within rangeTiles and lit enough: at full light it's seen as far as the range
// reaches, and the dimmer it is, the closer it has to be. A tile at light 0 is never seen, the viewer's own included,
// and nothing is seen with a range of 0.
func (x *Vision) Update(origin utils.Vec2i, rangeTiles int, isOpaque func(utils.Vec2i) bool, lightAt func(utils.Vec2i) uint8) {
	x.origin, x.rangeTiles = origin, rangeTiles
	side := 2*rangeTiles + 1
	if cap(x.visible) < side*side {
		x.visible = make([]bool, side*side)
	} else {
		x.visible = x.visible[:side*side]
		clear(x.visible)
	}
	if rangeTiles == 0 {
		return
	}

	fov.Compute(origin, rangeTiles, isOpaque, func(t utils.Vec2i) {
		if litEnough(t.X-origin.X, t.Y-origin.Y, lightAt(t), rangeTiles) {
			x.visible[x.index(t)] = true
		}
	})
}

// In integers, so that it's the same on every machine
func litEnough(dx, dy int, light uint8, rangeTiles int) bool {
	reach := int(light) * rangeTiles
	return light > 0 && (dx*dx+dy*dy)*data.MaxLight*data.MaxLight <= reach*reach
}

// The light around the viewer, kept with the vision so that working it out every tick doesn't allocate
func (x *Vision) Light() *LightMap {
	return &x.light
}

// The square of tiles CanSee answers for, as of the last update
func (x *Vision) VisibleArea() (origin utils.Vec2i, radius int) {
	return x.origin, x.rangeTiles
}

func (x *Vision) CanSee(t utils.Vec2i) bool {
	dx, dy := t.X-x.origin.X, t.Y-x.origin.Y
	if x.visible == nil || utils.AbsInt(dx) > x.rangeTiles || utils.AbsInt(dy) > x.rangeTiles {
		return false
	}
	return x.visible[x.index(t)]
}

func (x *Vision) index(t utils.Vec2i) int {
	side := 2*x.rangeTiles + 1
	return (t.Y-x.origin.Y+x.rangeTiles)*side + (t.X - x.origin.X + x.rangeTiles)
}

func (x Vision) Copy() Vision {
	return Vision{baseRangeTiles: x.baseRangeTiles}
}

func (x Vision) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Visions.addComponent(e, x.Copy())
}

type visionDto struct {
	BaseRangeTiles int
}

func (x Vision) toDto() (ComponentDto, error) {
	return &visionDto{BaseRangeTiles: x.baseRangeTiles}, nil
}

func (x visionDto) toComponent() (Component, error) {
	return NewVisionComponent(x.BaseRangeTiles), nil
}
