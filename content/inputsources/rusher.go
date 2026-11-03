package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

type RusherState uint8

const (
	// Stands in place
	Rusher_Idle RusherState = iota
	// Heads straight at its target, without pathfinding
	Rusher_Follow
)

// Saved with its state, so that a loaded rusher carries on as it was
type InputRusherParams struct {
	Target common.EntityId
	State  RusherState
}

func (InputRusherParams) IsInputParams() {}

func (p *InputRusherParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	return &c
}

// Follows its target while its vision sees the target's tile, and idles otherwise. Its vision is as of the last tick,
// since vision is worked out after inputs are handled.
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
		return towards(entityId, p.Target, ecsContainer)
	}
	return ecs.InputState{}, nil
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
