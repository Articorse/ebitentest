package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
)

// How far a body reaches past the tile it's in the middle of, up and left as negative numbers
type bodyBox struct {
	lo, hi utils.Vec2i
}

// What's kept for a tile: nothing yet, or one more than the speed it's walked at, with doorFlag set when the body would
// be over a door
const (
	unknownSpeed uint8 = 0
	doorFlag     uint8 = 0x80
)

// How fast a body that reaches lo to hi tiles past a tile walks standing in the middle of it: as fast as on that tile,
// or 0 where any tile it would take up can't be walked on. Also whether one of them is a door, which the body's faction
// decides it can go through or not. Kept for each chunk and size of body, since a path search asks for every tile it
// looks at, and the boxes round them overlap. The function remembers the last chunk it looked in, since a search's
// tiles are mostly in the same one.
func (cc *ChunkContainer) BodySpeeds(lo, hi utils.Vec2i) func(t utils.Vec2i) (speed int, door bool) {
	box := bodyBox{lo, hi}
	lastPos, version := utils.Vec2i{}, cc.bodyFitsVersion
	var last *chunk
	var known *[data.ChunkSize * data.ChunkSize]uint8
	return func(t utils.Vec2i) (int, bool) {
		pos := utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}
		if last == nil || pos != lastPos || version != cc.bodyFitsVersion {
			c, ok := cc.chunks[pos]
			if !ok {
				last = nil
				return 0, false
			}
			last, lastPos, version = c, pos, cc.bodyFitsVersion
			if c.bodyFits == nil {
				c.bodyFits = map[bodyBox]*[data.ChunkSize * data.ChunkSize]uint8{}
			}
			if known = c.bodyFits[box]; known == nil {
				known = &[data.ChunkSize * data.ChunkSize]uint8{}
				c.bodyFits[box] = known
			}
		}
		idx := last.tileIndex(t)
		if known[idx] == unknownSpeed {
			known[idx] = cc.bodySpeed(t, box)
		}
		return int(known[idx]&^doorFlag) - 1, known[idx]&doorFlag != 0
	}
}

func (cc *ChunkContainer) bodySpeed(t utils.Vec2i, box bodyBox) uint8 {
	door := uint8(0)
	for y := t.Y + box.lo.Y; y <= t.Y+box.hi.Y; y++ {
		for x := t.X + box.lo.X; x <= t.X+box.hi.X; x++ {
			n := utils.Vec2i{X: x, Y: y}
			if cc.WalkSpeedPercent(n) <= 0 {
				return 1
			}
			if _, ok := cc.DoorAt(n); ok {
				door = doorFlag
			}
		}
	}
	return uint8(cc.WalkSpeedPercent(t)+1) | door
}

// A body round a tile can reach into the chunks next to its own, so theirs go too
func (cc *ChunkContainer) forgetBodyFits(pos utils.Vec2i) {
	cc.bodyFitsVersion++
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if c, ok := cc.chunks[utils.Vec2i{X: pos.X + dx, Y: pos.Y + dy}]; ok {
				c.bodyFits = nil
			}
		}
	}
}
