package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
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
	// The hostile it's following, or last followed
	Target common.EntityId
	State  RusherState
	// The tile the target was on when it was last seen
	LastSeen utils.Vec2i

	walk
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

// Follows the closest hostile its vision sees, switching whenever another one is closer. Once it sees none, it heads for
// the tile it last saw its target on, and idles once it's there or as close as it can get. Its vision is as of the last
// tick, since vision is worked out after inputs are handled. It stands still while regions say it can't get to the
// target, and otherwise goes as close as a path gets it.
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

	var targetPos utils.Vec2f
	target, seesTarget := ecsContainer.NearestVisibleHostile(entityId)
	if seesTarget {
		if targetPos, err = ecsContainer.WorldPos(target); err != nil {
			return ecs.InputState{}, fmt.Errorf("error getting world position of rusher %d's target %d: %v", entityId, target, err)
		}
	}
	switch {
	case seesTarget:
		p.Target, p.State, p.LastSeen = target, Rusher_Follow, utils.WorldPosToTilePos(targetPos)
	case p.State == Rusher_Follow:
		p.State = Rusher_Search
	case p.State == Rusher_Search && utils.WorldPosToTilePos(selfPos) == p.LastSeen:
		p.State = Rusher_Idle
	}

	self, faction := utils.WorldPosToTilePos(selfPos), ecsContainer.FactionOf(entityId)
	switch p.State {
	case Rusher_Follow:
		if !ecsContainer.Terrain().Reachable(self, utils.WorldPosToTilePos(targetPos), faction) {
			p.path = nil
			return ecs.InputState{}, nil
		}
		input, _ := p.approach(selfPos, targetPos, ecsContainer.WalkSpeedOf(entityId), tick)
		return input, nil
	case Rusher_Search:
		input, ok := ecs.InputState{}, ecsContainer.Terrain().Reachable(self, p.LastSeen, faction)
		if ok {
			input, ok = p.approach(selfPos, utils.TilePosToWorldPos(p.LastSeen), ecsContainer.WalkSpeedOf(entityId), tick)
		}
		if !ok {
			p.State = Rusher_Idle
		}
		return input, nil
	}
	p.path = nil
	return ecs.InputState{}, nil
}
