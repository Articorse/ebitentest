package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
)

type Vision struct {
	radius int

	// What was visible at the last update, as a square around origin. It's derived from the world, so it isn't saved.
	origin        utils.Vec2i
	visibleRadius int
	visible       []bool
}

func NewVisionComponent(radius int) *Vision {
	return &Vision{radius: max(radius, 0)}
}

func (x *Vision) Radius() int {
	return x.radius
}

// Takes effect at the next update
func (x *Vision) SetRadius(radius int) {
	x.radius = max(radius, 0)
}

func (x *Vision) Update(origin utils.Vec2i, isOpaque func(utils.Vec2i) bool) {
	side := 2*x.radius + 1
	if cap(x.visible) < side*side {
		x.visible = make([]bool, side*side)
	} else {
		x.visible = x.visible[:side*side]
		clear(x.visible)
	}
	x.origin, x.visibleRadius = origin, x.radius

	fov.Compute(origin, x.radius, isOpaque, func(t utils.Vec2i) {
		x.visible[x.index(t)] = true
	})
}

// In pixels, around the viewer's exact position: fully clear within clearRadius, fading out up to outer. Kept far enough
// inside the computed radius that the circle only shows computed tiles, wherever in its tile the viewer stands.
func (x *Vision) CircleRadii() (clearRadius, outer float64) {
	outer = max(float64(x.visibleRadius)-data.VisionCircleMarginTiles, data.VisionCircleFadeTiles) * data.TileSize
	return outer - data.VisionCircleFadeTiles*data.TileSize, outer
}

func (x *Vision) CanSee(t utils.Vec2i) bool {
	dx, dy := t.X-x.origin.X, t.Y-x.origin.Y
	if x.visible == nil || utils.AbsInt(dx) > x.visibleRadius || utils.AbsInt(dy) > x.visibleRadius {
		return false
	}
	return x.visible[x.index(t)]
}

func (x *Vision) index(t utils.Vec2i) int {
	side := 2*x.visibleRadius + 1
	return (t.Y-x.origin.Y+x.visibleRadius)*side + (t.X - x.origin.X + x.visibleRadius)
}

func (x Vision) Copy() Vision {
	return Vision{radius: x.radius}
}

func (x Vision) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Visions.addComponent(e, x.Copy())
}

type visionDto struct {
	Radius int
}

func (x Vision) toDto() (ComponentDto, error) {
	return &visionDto{Radius: x.radius}, nil
}

func (x visionDto) toComponent() (Component, error) {
	return NewVisionComponent(x.Radius), nil
}
