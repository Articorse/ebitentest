package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
)

func (cc *ChunkContainer) WalkSpeedPercent(t utils.Vec2i) int {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0
	}
	if c.walkSpeeds == nil {
		c.walkSpeeds = cc.chunkWalkSpeeds(c)
	}
	return int(c.walkSpeeds[c.tileIndex(t)])
}

// A feature that's walked through, such as a torch, doesn't slow walking, and nor does a door, which opens for whoever
// would walk through it
func (cc *ChunkContainer) chunkWalkSpeeds(c *chunk) *[data.ChunkSize * data.ChunkSize]uint8 {
	speeds := &[data.ChunkSize * data.ChunkSize]uint8{}
	for i, p := range c.props {
		if p.passable {
			speeds[i] = 100 - p.slowdownPercent
		}
	}
	for t := range c.features {
		if _, walkThrough, ok := cc.featureCollider(c, t); ok && !walkThrough && cc.featureAtlas[c.features[t]].door == nil {
			speeds[c.tileIndex(t)] = 0
		}
	}
	return speeds
}
