package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type animationManager struct{}

func NewAnimationComponent(
	sheetAssetTag common.ImageAssetTag,
	stateFrames map[AnimationState][]AnimationFrame,
) (*animation, error) {
	return &animation{
		sheetAssetTag: sheetAssetTag,
		stateFrames:   stateFrames,
	}, nil
}

func (animationManager) GetState(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (AnimationState, error) {
	animComp, err := ecsContainer.Animations.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	return animComp.frameState.AnimationState, nil
}

func (animationManager) SetState(
	e common.EntityId,
	newState AnimationState,
	ecsContainer *ECSContainer,
) error {
	animComp, err := ecsContainer.Animations.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite component of entity %d: %v", e, err)
	}

	if _, ok := animComp.stateFrames[newState]; !ok {
		return fmt.Errorf("animation state %d not found for entity %d", newState, e)
	}

	animComp.frameState.AnimationState = newState
	animComp.frameState.CurrentIdx = 0
	animComp.frameState.CounterMs = 0
	sprComp.subImageIdx = 0

	return nil
}

func (animationManager) SetQueuedStateIfNone(
	e common.EntityId,
	newState AnimationState,
	ecsContainer *ECSContainer,
) error {
	animComp, err := ecsContainer.Animations.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	if _, ok := animComp.stateFrames[newState]; !ok {
		return fmt.Errorf("animation state %d not found for entity %d", newState, e)
	}

	if animComp.queuedState == nil {
		animComp.queuedState = &newState
	}

	return nil
}

func (animationManager) Tick(
	e common.EntityId,
	ecsContainer *ECSContainer,
) error {
	animComp, err := ecsContainer.Animations.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite component of entity %d: %v", e, err)
	}

	currentState := animComp.frameState.AnimationState
	currentIdx := animComp.frameState.CurrentIdx
	// currentIdx := sprComp.subImageIdx

	frames, ok := animComp.stateFrames[currentState]
	if !ok {
		return fmt.Errorf("animation state %d not found for entity %d", currentState, e)
	}

	if int(currentIdx) >= len(frames) {
		return fmt.Errorf("current frame index %d out of bounds for animation state %d of entity %d", currentIdx, currentState, e)
	}

	frame := frames[currentIdx]

	animComp.frameState.CounterMs += data.TickMs

	if animComp.frameState.CounterMs >= frame.DurationMs {
		if animComp.queuedState != nil {
			animComp.frameState.AnimationState = *animComp.queuedState
			animComp.queuedState = nil
		} else {
			animComp.frameState.CounterMs = 0
			animComp.frameState.CurrentIdx = (animComp.frameState.CurrentIdx + 1) % uint16(len(frames))
			sprComp.subImageIdx = int(animComp.stateFrames[currentState][currentIdx].FrameIdx)
		}
	}

	return nil
}
