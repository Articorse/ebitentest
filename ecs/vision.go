package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
)

// Sees the tiles in line of sight that are lit enough for how far away they are
type Vision struct {
	// What was visible at the last update, as a square around origin. It's derived from the world, so it isn't saved.
	origin  utils.Vec2i
	visible []bool
	light   LightMap
}

func NewVisionComponent() *Vision {
	return &Vision{}
}

// A tile is seen if it's in line of sight within data.VisionMaxRangeTiles and lit enough: at full light it's seen as
// far as that range reaches, and the dimmer it is, the closer it has to be. A tile at light 0 is never seen, the
// viewer's own included.
func (x *Vision) Update(origin utils.Vec2i, isOpaque func(utils.Vec2i) bool, lightAt func(utils.Vec2i) uint8) {
	const side = 2*data.VisionMaxRangeTiles + 1
	if cap(x.visible) < side*side {
		x.visible = make([]bool, side*side)
	} else {
		x.visible = x.visible[:side*side]
		clear(x.visible)
	}
	x.origin = origin

	fov.Compute(origin, data.VisionMaxRangeTiles, isOpaque, func(t utils.Vec2i) {
		if litEnough(t.X-origin.X, t.Y-origin.Y, lightAt(t)) {
			x.visible[x.index(t)] = true
		}
	})
}

// In integers, so that it's the same on every machine
func litEnough(dx, dy int, light uint8) bool {
	reach := int(light) * data.VisionMaxRangeTiles
	return light > 0 && (dx*dx+dy*dy)*data.MaxLight*data.MaxLight <= reach*reach
}

// The light around the viewer, kept with the vision so that working it out every tick doesn't allocate
func (x *Vision) Light() *LightMap {
	return &x.light
}

// The square of tiles CanSee answers for, as of the last update
func (x *Vision) VisibleArea() (origin utils.Vec2i, radius int) {
	return x.origin, data.VisionMaxRangeTiles
}

func (x *Vision) CanSee(t utils.Vec2i) bool {
	dx, dy := t.X-x.origin.X, t.Y-x.origin.Y
	if x.visible == nil || utils.AbsInt(dx) > data.VisionMaxRangeTiles || utils.AbsInt(dy) > data.VisionMaxRangeTiles {
		return false
	}
	return x.visible[x.index(t)]
}

func (x *Vision) index(t utils.Vec2i) int {
	const side = 2*data.VisionMaxRangeTiles + 1
	return (t.Y-x.origin.Y+data.VisionMaxRangeTiles)*side + (t.X - x.origin.X + data.VisionMaxRangeTiles)
}

func (x Vision) Copy() Vision {
	return Vision{}
}

func (x Vision) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Visions.addComponent(e, x.Copy())
}

type visionDto struct{}

func (x Vision) toDto() (ComponentDto, error) {
	return &visionDto{}, nil
}

func (x visionDto) toComponent() (Component, error) {
	return NewVisionComponent(), nil
}
