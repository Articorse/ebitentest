package inputsources

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/pathfinding"
	"ebittest/utils"
	"fmt"
	"slices"
)

type RusherState uint8

const (
	// Stands in place
	Rusher_Idle RusherState = iota
	// Heads for its target
	Rusher_Follow
	// Heads for where it last saw its target, after losing sight of it
	Rusher_Search
)

// Saved with its state, so that a loaded rusher carries on as it was. Its path isn't saved, since it's found again.
type InputRusherParams struct {
	Target common.EntityId
	State  RusherState
	// The tile the target was on when it was last seen
	LastSeen utils.Vec2i

	// From the tile it's on to the tile goal was on when the path was found
	path []utils.Vec2i
	goal utils.Vec2i
}

func (InputRusherParams) IsInputParams() {}

func (p *InputRusherParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.path = slices.Clone(p.path)
	return &c
}

func (p *InputRusherParams) CurrentPath() []utils.Vec2i {
	return p.path
}

// Follows its target while its vision sees the target's tile. Once it loses sight of it, it heads for the tile it last
// saw it on, and idles once it's there or there's no way there. Its vision is as of the last tick, since vision is
// worked out after inputs are handled. It stands still while no path reaches the target.
func RusherInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	p, ok := params.(*InputRusherParams)
	if !ok || p == nil {
		return ecs.InputState{}, fmt.Errorf("rusher input source called with invalid params: %T", params)
	}

	selfPos, err := ecsContainer.WorldPos(entityId)
	if err != nil {
		return ecs.InputState{}, fmt.Errorf("error getting world position of rusher %d: %v", entityId, err)
	}

	targetPos, seesTarget := seenPos(entityId, p.Target, ecsContainer)
	switch {
	case seesTarget:
		p.State, p.LastSeen = Rusher_Follow, utils.WorldPosToTilePos(targetPos)
	case p.State == Rusher_Follow:
		p.State = Rusher_Search
	case p.State == Rusher_Search && utils.WorldPosToTilePos(selfPos) == p.LastSeen:
		p.State = Rusher_Idle
	}

	switch p.State {
	case Rusher_Follow:
		input, _ := p.moveTo(selfPos, targetPos, ecsContainer.Terrain())
		return input, nil
	case Rusher_Search:
		input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.LastSeen), ecsContainer.Terrain())
		if !ok {
			p.State = Rusher_Idle
		}
		return input, nil
	}
	p.path = nil
	return ecs.InputState{}, nil
}

// Straight at the destination while nothing's in the way, and along a path to it otherwise. The path is found again
// when the destination moves to another tile, or something pushes the rusher off it. False, and standing still, when no
// path reaches it.
func (p *InputRusherParams) moveTo(selfPos, dest utils.Vec2f, terrain ecs.Terrain) (ecs.InputState, bool) {
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

// Where the target is, if the viewer's vision sees its tile. False for a target that no longer exists, or a viewer
// without vision.
func seenPos(viewer, target common.EntityId, ecsContainer *ecs.ECSContainer) (utils.Vec2f, bool) {
	vision, ok := ecsContainer.Visions.Get(viewer)
	if !ok {
		return utils.Vec2f{}, false
	}
	pos, err := ecsContainer.WorldPos(target)
	return pos, err == nil && vision.CanSee(utils.WorldPosToTilePos(pos))
}
