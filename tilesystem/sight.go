package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
)

// Looks each chunk up once rather than once per tile, which is most of the cost of asking tile by tile over the large
// squares light and vision work on
func (cc *ChunkContainer) TileFlags(from utils.Vec2i, w, h int, opaque, openToSky []bool) {
	to := utils.Vec2i{X: from.X + w - 1, Y: from.Y + h - 1}
	for cy := floorDiv(from.Y, data.ChunkSize); cy <= floorDiv(to.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(from.X, data.ChunkSize); cx <= floorDiv(to.X, data.ChunkSize); cx++ {
			c := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			x0, x1 := max(from.X, cx*data.ChunkSize), min(to.X, (cx+1)*data.ChunkSize-1)
			y0, y1 := max(from.Y, cy*data.ChunkSize), min(to.Y, (cy+1)*data.ChunkSize-1)
			for y := y0; y <= y1; y++ {
				row := (y - from.Y) * w
				for x := x0; x <= x1; x++ {
					i := row + x - from.X
					if c == nil {
						setFlag(opaque, i, true)
						setFlag(openToSky, i, false)
						continue
					}
					local := c.tileIndex(utils.Vec2i{X: x, Y: y})
					setFlag(opaque, i, cc.isOpaqueTileDef(c.tiles[local]) || cc.opaqueFeaturesOf(c)[utils.Vec2i{X: x, Y: y}])
					setFlag(openToSky, i, c.roofs[local] == data.Roof_None)
				}
			}
		}
	}
}

func setFlag(flags []bool, i int, v bool) {
	if flags != nil {
		flags[i] = v
	}
}

// Tiles that aren't in the atlas count as opaque, like those outside the active chunks
func (cc *ChunkContainer) isOpaqueTileDef(tile data.TileEnum) bool {
	return int(tile) >= len(cc.opaqueTiles) || cc.opaqueTiles[tile]
}

// Goes up whenever a tile or feature changes, or a chunk comes into or leaves the simulation, which is everything that
// sight and light depend on besides the time of day and lights carried around
func (cc *ChunkContainer) SightVersion() uint64 {
	return cc.sightVersion
}
