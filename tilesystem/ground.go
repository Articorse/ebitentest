package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"maps"
	"slices"
)

func (cc *ChunkContainer) GroundStack(t utils.Vec2i) (ecs.GroundStack, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return ecs.GroundStack{}, false
	}
	s, ok := c.ground[t]
	return s, ok
}

func (cc *ChunkContainer) GroundStacksIn(first, last utils.Vec2i) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for cy := floorDiv(first.Y, data.ChunkSize); cy <= floorDiv(last.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(first.X, data.ChunkSize); cx <= floorDiv(last.X, data.ChunkSize); cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok {
				continue
			}
			for t := range c.ground {
				if t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y {
					tiles = append(tiles, t)
				}
			}
		}
	}
	// Map order differs from run to run, and what's done with them mustn't
	slices.SortFunc(tiles, func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return tiles
}

func (cc *ChunkContainer) SetGroundStack(t utils.Vec2i, s ecs.GroundStack) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}
	if old, had := c.ground[t]; had && old.Unique == nil && cc.IsHarvestItem(old.Kind) && (s.Unique != nil || s.Kind != old.Kind || s.Amount < old.Amount) {
		cc.noteHarvested(t)
	}
	if s.Unique == nil && s.Amount <= 0 {
		delete(c.ground, t)
	} else {
		c.ground[t] = s
	}
	c.terrainModified = true
	return true
}

type groundStackDto struct {
	X, Y  int
	Stack ecs.GroundStack
}

// In tile order, so that saving the same chunk twice writes the same file
func groundDtos(c *chunk) []groundStackDto {
	dtos := make([]groundStackDto, 0, len(c.ground))
	for _, t := range slices.SortedFunc(maps.Keys(c.ground), func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	}) {
		dtos = append(dtos, groundStackDto{X: t.X, Y: t.Y, Stack: c.ground[t]})
	}
	return dtos
}

func groundFromDtos(dtos []groundStackDto) map[utils.Vec2i]ecs.GroundStack {
	ground := make(map[utils.Vec2i]ecs.GroundStack, len(dtos))
	for _, d := range dtos {
		ground[utils.Vec2i{X: d.X, Y: d.Y}] = d.Stack
	}
	return ground
}
