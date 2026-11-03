package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
)

// Calls lit for every feature in the active chunks whose light reaches the square between min and max
func (cc *ChunkContainer) LightSources(min, max utils.Vec2i, lit func(t utils.Vec2i, level uint8, rangeTiles int)) {
	if cc.maxLightRange == 0 {
		return
	}

	reachMin := utils.Vec2i{X: min.X - cc.maxLightRange, Y: min.Y - cc.maxLightRange}
	reachMax := utils.Vec2i{X: max.X + cc.maxLightRange, Y: max.Y + cc.maxLightRange}
	minChunk := utils.Vec2i{X: floorDiv(reachMin.X, data.ChunkSize), Y: floorDiv(reachMin.Y, data.ChunkSize)}
	maxChunk := utils.Vec2i{X: floorDiv(reachMax.X, data.ChunkSize), Y: floorDiv(reachMax.Y, data.ChunkSize)}

	for cy := minChunk.Y; cy <= maxChunk.Y; cy++ {
		for cx := minChunk.X; cx <= maxChunk.X; cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok {
				continue
			}

			for _, t := range cc.chunkLights(c) {
				def := cc.featureAtlas[c.features[t]]
				r := def.lightRangeTiles
				if t.X+r < min.X || t.X-r > max.X || t.Y+r < min.Y || t.Y-r > max.Y {
					continue
				}
				lit(t, def.light, r)
			}
		}
	}
}

func (cc *ChunkContainer) chunkLights(c *chunk) []utils.Vec2i {
	if c.lightsIndexed {
		return c.lights
	}

	c.lights = c.lights[:0]
	for t, f := range c.features {
		if cc.featureAtlas[f].light > 0 {
			c.lights = append(c.lights, t)
		}
	}
	c.lightsIndexed = true
	return c.lights
}
