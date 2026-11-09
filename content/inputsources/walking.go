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
	// The tick from which it can look for a path again, after one failed to reach where it's going
	retryTick uint64
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
// when the destination moves to another tile, something pushes the walker off it, or something's built on it. False,
// and standing still, when no path reaches it.
func (p *walk) moveTo(selfPos, dest utils.Vec2f, walkSpeed func(utils.Vec2i) int) (ecs.InputState, bool) {
	if clearWalk(selfPos, dest, walkSpeed) {
		p.path = nil
		return heading(selfPos, dest), true
	}

	from, goal := utils.WorldPosToTilePos(selfPos), utils.WorldPosToTilePos(dest)
	on := slices.Index(p.path, from)
	blocked := on != -1 && slices.ContainsFunc(p.path[on:], func(t utils.Vec2i) bool { return walkSpeed(t) <= 0 })
	if on == -1 || p.goal != goal || blocked {
		path, ok := pathfinding.Find(from, goal, data.PathMaxExpandedTiles, walkSpeed)
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
	return p.alongPath(selfPos, walkSpeed), true
}

// As moveTo, but when no path reaches the destination, to the closest it can get, where it waits. A failed search
// isn't tried again for data.PathRetryTicks, keeping to the path it has, so that chasing what's out of reach doesn't
// search every tick. False once it's as close as it can get.
func (p *walk) approach(selfPos, dest utils.Vec2f, walkSpeed func(utils.Vec2i) int, tick uint64) (ecs.InputState, bool) {
	if clearWalk(selfPos, dest, walkSpeed) {
		p.path = nil
		return heading(selfPos, dest), true
	}

	from, goal := utils.WorldPosToTilePos(selfPos), utils.WorldPosToTilePos(dest)
	on := slices.Index(p.path, from)
	blocked := on != -1 && slices.ContainsFunc(p.path[on:], func(t utils.Vec2i) bool { return walkSpeed(t) <= 0 })
	short := len(p.path) > 0 && p.path[len(p.path)-1] != goal
	if (on == -1 || p.goal != goal || blocked || short) && tick >= p.retryTick {
		path, reached := pathfinding.FindNearest(from, goal, data.PathMaxExpandedTiles, walkSpeed)
		if !reached {
			p.retryTick = tick + data.PathRetryTicks
		}
		p.path, p.goal, on = append([]utils.Vec2i{from}, path...), goal, 0
	}
	if on == -1 || blocked {
		p.path = nil
		return ecs.InputState{}, false
	}
	p.path = p.path[on:]

	if len(p.path) < 2 {
		if p.path[0] != goal {
			return ecs.InputState{}, false
		}
		return heading(selfPos, dest), true
	}
	return p.alongPath(selfPos, walkSpeed), true
}

// Towards the furthest tile ahead it can head straight to
func (p *walk) alongPath(selfPos utils.Vec2f, walkSpeed func(utils.Vec2i) int) ecs.InputState {
	next := p.path[1]
	for _, t := range p.path[2:min(len(p.path), data.PathLookaheadTiles+1)] {
		if !clearWalk(selfPos, utils.TilePosToWorldPos(t), walkSpeed) {
			break
		}
		next = t
	}
	return heading(selfPos, utils.TilePosToWorldPos(next))
}

// Every tile between the two, leaving out the ones they're on, can be walked at full speed, so that slower ground is
// left to paths to weigh up. With the walker's own walkSpeed, which keeps a big body's room round each tile, that sweeps
// the whole body along the line.
func clearWalk(a, b utils.Vec2f, walkSpeed func(utils.Vec2i) int) bool {
	tiles := utils.TilesAlongSegment(a, b)
	if len(tiles) <= 2 {
		return true
	}
	for _, t := range tiles[1 : len(tiles)-1] {
		if walkSpeed(t) < 100 {
			return false
		}
	}
	return true
}

func heading(from, to utils.Vec2f) ecs.InputState {
	dir := to.Subtract(from).Normalized()
	return ecs.InputState{Analog1X: dir.X, Analog1Y: dir.Y}
}
