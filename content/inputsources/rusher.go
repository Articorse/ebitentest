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
	// Heads straight at its target, without pathfinding
	Rusher_Follow
)

// Saved with its state, so that a loaded rusher carries on as it was. Its path isn't saved, since it's found again.
type InputRusherParams struct {
	Target common.EntityId
	State  RusherState

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

// Follows its target while its vision sees the target's tile, and idles otherwise. Its vision is as of the last tick,
// since vision is worked out after inputs are handled. It stands still while no path reaches the target.
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

	seesTarget := canSee(entityId, p.Target, ecsContainer)
	switch p.State {
	case Rusher_Idle:
		if seesTarget {
			p.State = Rusher_Follow
		}
	case Rusher_Follow:
		if !seesTarget {
			p.State = Rusher_Idle
		}
	}

	if p.State == Rusher_Follow {
		return p.follow(entityId, ecsContainer)
	}
	p.path = nil
	return ecs.InputState{}, nil
}

// Straight at the target while nothing's in the way, and along a path to it otherwise. The path is found again when the
// target moves to another tile, or something pushes the rusher off it.
func (p *InputRusherParams) follow(self common.EntityId, ecsContainer *ecs.ECSContainer) (ecs.InputState, error) {
	selfPos, err := ecsContainer.WorldPos(self)
	if err != nil {
		return ecs.InputState{}, fmt.Errorf("error getting world position of rusher %d: %v", self, err)
	}
	targetPos, err := ecsContainer.WorldPos(p.Target)
	if err != nil {
		return ecs.InputState{}, fmt.Errorf("error getting world position of rusher %d's target %d: %v", self, p.Target, err)
	}
	terrain := ecsContainer.Terrain()

	if clearWalk(selfPos, targetPos, terrain) {
		p.path = nil
		return heading(selfPos, targetPos), nil
	}

	from, goal := utils.WorldPosToTilePos(selfPos), utils.WorldPosToTilePos(targetPos)
	on := slices.Index(p.path, from)
	if on == -1 || p.goal != goal {
		path, ok := pathfinding.Find(from, goal, data.PathMaxExpandedTiles, terrain.WalkSpeedPercent)
		if !ok {
			p.path = nil
			return ecs.InputState{}, nil
		}
		p.path, p.goal, on = append([]utils.Vec2i{from}, path...), goal, 0
	}
	p.path = p.path[on:]

	if len(p.path) < 2 {
		return heading(selfPos, targetPos), nil
	}
	next := p.path[1]
	for _, t := range p.path[2:min(len(p.path), data.PathLookaheadTiles+1)] {
		if !clearWalk(selfPos, utils.TilePosToWorldPos(t), terrain) {
			break
		}
		next = t
	}
	return heading(selfPos, utils.TilePosToWorldPos(next)), nil
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

// False for a target that no longer exists, or a viewer without vision
func canSee(viewer, target common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	vision, ok := ecsContainer.Visions.Get(viewer)
	if !ok {
		return false
	}
	pos, err := ecsContainer.WorldPos(target)
	return err == nil && vision.CanSee(utils.WorldPosToTilePos(pos))
}
