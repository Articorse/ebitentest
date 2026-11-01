package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"math"
)

// Moves a tethered entity back inside its anchor's loaded area, as close as possible to where it was.
// Returns false if e isn't tethered to a live chunkloader, in which case it's resolved like any other entity.
func (cc *ChunkContainer) relocateTethered(e common.EntityId, ecsCont *ecs.ECSContainer) bool {
	if !ecsCont.Tethers.HasComponent(e) {
		return false
	}

	tm := ecsCont.TransformManager
	vm := ecsCont.VelocityManager
	hm := ecsCont.HierarchyManager

	anchor, err := ecsCont.TetherManager.GetAnchor(e, ecsCont)
	if err != nil || !ecsCont.ChunkLoaders.HasComponent(anchor) {
		return false
	}

	anchorPos, err := tm.GetWorldPos(anchor, ecsCont)
	if err != nil {
		log.Printf("error getting world position of anchor %d of entity %d: %v", anchor, e, err)
		return false
	}

	radius, err := ecsCont.ChunkLoaderManager.GetRadius(anchor, ecsCont)
	if err != nil {
		log.Printf("error getting chunkloader radius of anchor %d of entity %d: %v", anchor, e, err)
		return false
	}

	ePos, err := tm.GetWorldPos(e, ecsCont)
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

	eTile := worldPosToTilePos(ePos)
	start := utils.Vec2i{
		X: min(max(eTile.X, minTile.X), maxTile.X),
		Y: min(max(eTile.Y, minTile.Y), maxTile.Y),
	}

	targetPos := anchorPos
	if tile, found := cc.findClosestPassableTile(start, minTile, maxTile); found {
		targetPos = tilePosToWorldPos(tile)
	}

	if err := hm.Detach(e, ecsCont); err != nil {
		log.Printf("error detaching tethered entity %d: %v", e, err)
	}

	if err := tm.SetWorldPos(e, targetPos, ecsCont); err != nil {
		log.Printf("error moving tethered entity %d: %v", e, err)
		return false
	}

	if ecsCont.Velocities.HasComponent(e) {
		if err := vm.SetLocalVector(e, utils.Vec2f{}, ecsCont); err != nil {
			log.Printf("error resetting velocity of tethered entity %d: %v", e, err)
		}
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

// Tiles are centered on multiples of the tile size
func worldPosToTilePos(pos utils.Vec2f) utils.Vec2i {
	return utils.Vec2i{
		X: int(math.Floor(pos.X/data.TileSize + 0.5)),
		Y: int(math.Floor(pos.Y/data.TileSize + 0.5)),
	}
}

func tilePosToWorldPos(t utils.Vec2i) utils.Vec2f {
	return utils.Vec2f{X: float64(t.X * data.TileSize), Y: float64(t.Y * data.TileSize)}
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
