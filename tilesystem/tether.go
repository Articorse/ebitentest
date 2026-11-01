package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

// Moves a tethered entity back inside its anchor's loaded area, as close as possible to where it was.
// Returns false if e isn't tethered to a live chunkloader, in which case it's resolved like any other entity.
func (cc *ChunkContainer) relocateTethered(e common.EntityId, ecsCont *ecs.ECSContainer) bool {
	tether, ok := ecsCont.Tethers.Get(e)
	if !ok {
		return false
	}

	anchor := tether.Anchor()
	loader, ok := ecsCont.ChunkLoaders.Get(anchor)
	if !ok {
		return false
	}

	anchorPos, err := ecsCont.WorldPos(anchor)
	if err != nil {
		log.Printf("error getting world position of anchor %d of entity %d: %v", anchor, e, err)
		return false
	}

	radius := loader.Radius()

	ePos, err := ecsCont.WorldPos(e)
	if err != nil {
		log.Printf("error getting world position of tethered entity %d: %v", e, err)
		return false
	}

	center := utils.WorldPosToChunkGridPos(anchorPos)
	minTile := utils.Vec2i{
		X: (center.X-radius)*data.ChunkSize + data.TetherBorderTiles,
		Y: (center.Y-radius)*data.ChunkSize + data.TetherBorderTiles,
	}
	maxTile := utils.Vec2i{
		X: (center.X+radius+1)*data.ChunkSize - 1 - data.TetherBorderTiles,
		Y: (center.Y+radius+1)*data.ChunkSize - 1 - data.TetherBorderTiles,
	}

	eTile := utils.WorldPosToTilePos(ePos)
	start := utils.Vec2i{
		X: min(max(eTile.X, minTile.X), maxTile.X),
		Y: min(max(eTile.Y, minTile.Y), maxTile.Y),
	}

	targetPos := anchorPos
	if tile, found := cc.findClosestPassableTile(start, minTile, maxTile); found {
		targetPos = utils.TilePosToWorldPos(tile)
	}

	if err := ecsCont.Detach(e); err != nil {
		log.Printf("error detaching tethered entity %d: %v", e, err)
	}

	if err := ecsCont.SetWorldPos(e, targetPos); err != nil {
		log.Printf("error moving tethered entity %d: %v", e, err)
		return false
	}

	if v, ok := ecsCont.Velocities.Get(e); ok {
		v.SetLocalVector(utils.Vec2f{})
	}

	return true
}

// Searches square rings around start, so that the search stays bounded and never causes a hitch
func (cc *ChunkContainer) findClosestPassableTile(start, minTile, maxTile utils.Vec2i) (utils.Vec2i, bool) {
	for ring := 0; ring <= data.TetherSearchRadiusTiles; ring++ {
		best := utils.Vec2i{}
		bestDist := -1

		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if max(utils.AbsInt(dx), utils.AbsInt(dy)) != ring {
					continue
				}

				t := utils.Vec2i{X: start.X + dx, Y: start.Y + dy}
				if t.X < minTile.X || t.X > maxTile.X || t.Y < minTile.Y || t.Y > maxTile.Y {
					continue
				}

				if !cc.isTilePassable(t) {
					continue
				}

				d := dx*dx + dy*dy
				if bestDist == -1 || d < bestDist {
					best = t
					bestDist = d
				}
			}
		}

		if bestDist != -1 {
			return best, true
		}
	}

	return utils.Vec2i{}, false
}

func (cc *ChunkContainer) isTilePassable(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	return ok && def.passable
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
