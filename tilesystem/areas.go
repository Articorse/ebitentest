package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"slices"
)

// Where colonists look for work, and what work. False outside the active chunks.
func (cc *ChunkContainer) InArea(area data.AreaEnum, t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok || area >= data.AreaCount || c.areas[area] == nil {
		return false
	}
	i := c.tileIndex(t)
	return c.areas[area][i/64]&(1<<(i%64)) != 0
}

// Puts the tiles from first to last that are in the simulation in the area, or takes them out of it
func (cc *ChunkContainer) SetArea(area data.AreaEnum, first, last utils.Vec2i, in bool) {
	if area == data.Area_None || area >= data.AreaCount {
		return
	}
	for cy := floorDiv(first.Y, data.ChunkSize); cy <= floorDiv(last.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(first.X, data.ChunkSize); cx <= floorDiv(last.X, data.ChunkSize); cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok || (c.areas[area] == nil && !in) {
				continue
			}
			if c.areas[area] == nil {
				c.areas[area] = new([data.ChunkSize * data.ChunkSize / 64]uint64)
			}
			bits := c.areas[area]
			for y := max(first.Y, cy*data.ChunkSize); y <= min(last.Y, (cy+1)*data.ChunkSize-1); y++ {
				for x := max(first.X, cx*data.ChunkSize); x <= min(last.X, (cx+1)*data.ChunkSize-1); x++ {
					i := c.tileIndex(utils.Vec2i{X: x, Y: y})
					if in {
						bits[i/64] |= 1 << (i % 64)
					} else {
						bits[i/64] &^= 1 << (i % 64)
					}
				}
			}
			if !in && !slices.ContainsFunc(bits[:], func(w uint64) bool { return w != 0 }) {
				c.areas[area] = nil
			}
			c.terrainModified = true
		}
	}
}

// Indexed by area, with nil for an area the chunk has none of, which is most of them
func areasDto(c *chunk) [][]uint64 {
	areas := make([][]uint64, data.AreaCount)
	for a, bits := range c.areas {
		if bits != nil {
			areas[a] = slices.Clone(bits[:])
		}
	}
	return areas
}

func areasFromDto(dto [][]uint64) [data.AreaCount]*[data.ChunkSize * data.ChunkSize / 64]uint64 {
	var areas [data.AreaCount]*[data.ChunkSize * data.ChunkSize / 64]uint64
	for a, words := range dto {
		if a == int(data.Area_None) || a >= int(data.AreaCount) || len(words) != data.ChunkSize*data.ChunkSize/64 {
			continue
		}
		areas[a] = new([data.ChunkSize * data.ChunkSize / 64]uint64)
		copy(areas[a][:], words)
	}
	return areas
}
