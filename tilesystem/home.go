package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"slices"
)

// Where colonists look for work. False outside the active chunks.
func (cc *ChunkContainer) IsHome(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok || c.home == nil {
		return false
	}
	i := c.tileIndex(t)
	return c.home[i/64]&(1<<(i%64)) != 0
}

// Makes the tiles from first to last that are in the simulation home, or not
func (cc *ChunkContainer) SetHome(first, last utils.Vec2i, home bool) {
	for cy := floorDiv(first.Y, data.ChunkSize); cy <= floorDiv(last.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(first.X, data.ChunkSize); cx <= floorDiv(last.X, data.ChunkSize); cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok || (c.home == nil && !home) {
				continue
			}
			if c.home == nil {
				c.home = new([data.ChunkSize * data.ChunkSize / 64]uint64)
			}
			for y := max(first.Y, cy*data.ChunkSize); y <= min(last.Y, (cy+1)*data.ChunkSize-1); y++ {
				for x := max(first.X, cx*data.ChunkSize); x <= min(last.X, (cx+1)*data.ChunkSize-1); x++ {
					i := c.tileIndex(utils.Vec2i{X: x, Y: y})
					if home {
						c.home[i/64] |= 1 << (i % 64)
					} else {
						c.home[i/64] &^= 1 << (i % 64)
					}
				}
			}
			if !home && !slices.ContainsFunc(c.home[:], func(w uint64) bool { return w != 0 }) {
				c.home = nil
			}
			c.terrainModified = true
		}
	}
}

// Nil for a chunk with no home, which is most of them
func homeDto(c *chunk) []uint64 {
	if c.home == nil {
		return nil
	}
	return slices.Clone(c.home[:])
}

func homeFromDto(words []uint64) *[data.ChunkSize * data.ChunkSize / 64]uint64 {
	if len(words) != data.ChunkSize*data.ChunkSize/64 {
		return nil
	}
	home := new([data.ChunkSize * data.ChunkSize / 64]uint64)
	copy(home[:], words)
	return home
}
