package inputsources

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/pathfinding"
	"ebittest/utils"
	"slices"
)

// A path being walked, which isn't saved, since it's found again
type walk struct {
	// From the tile it's on to the tile goal was on when the path was found
	path []utils.Vec2i
	goal utils.Vec2i
}

func (p *walk) CurrentPath() []utils.Vec2i {
	return p.path
}

// Takes a path found elsewhere, from the tile after from, to walk to its last tile
func (p *walk) follow(from utils.Vec2i, path []utils.Vec2i) {
	p.path, p.goal = append([]utils.Vec2i{from}, path...), from
	if len(path) > 0 {
		p.goal = path[len(path)-1]
	}
}

// Straight at the destination while nothing's in the way, and along a path to it otherwise. The path is found again
// when the destination moves to another tile, or something pushes the walker off it. False, and standing still, when no
// path reaches it.
func (p *walk) moveTo(selfPos, dest utils.Vec2f, terrain ecs.Terrain) (ecs.InputState, bool) {
	if clearWalk(selfPos, dest, terrain) {
		p.path = nil
		return heading(selfPos, dest), true
	}

	from, goal := utils.WorldPosToTilePos(selfPos), utils.WorldPosToTilePos(dest)
	on := slices.Index(p.path, from)
	if on == -1 || p.goal != goal {
		path, ok := pathfinding.Find(from, goal, data.PathMaxExpandedTiles, terrain.WalkSpeedPercent)
		if !ok {
			p.path = nil
			return ecs.InputState{}, false
		}
		p.path, p.goal, on = append([]utils.Vec2i{from}, path...), goal, 0
	}
	p.path = p.path[on:]

	if len(p.path) < 2 {
		return heading(selfPos, dest), true
	}
	next := p.path[1]
	for _, t := range p.path[2:min(len(p.path), data.PathLookaheadTiles+1)] {
		if !clearWalk(selfPos, utils.TilePosToWorldPos(t), terrain) {
			break
		}
		next = t
	}
	return heading(selfPos, utils.TilePosToWorldPos(next)), true
}

// Every tile between the two, leaving out the ones they're on, can be walked at full speed, so that slower ground is
// left to paths to weigh up
func clearWalk(a, b utils.Vec2f, terrain ecs.Terrain) bool {
	tiles := utils.TilesAlongSegment(a, b)
	if len(tiles) <= 2 {
		return true
	}
	for _, t := range tiles[1 : len(tiles)-1] {
		if terrain.WalkSpeedPercent(t) < 100 {
			return false
		}
	}
	return true
}

func heading(from, to utils.Vec2f) ecs.InputState {
	dir := to.Subtract(from).Normalized()
	return ecs.InputState{Analog1X: dir.X, Analog1Y: dir.Y}
}
