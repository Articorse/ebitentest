package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
)

type AnimationState uint16

const (
	Anim_Idle AnimationState = iota
	Anim_Use
	Anim_Jump
)

type AnimationFrame struct {
	FrameIdx   uint16
	DurationMs uint64
}

type FrameState struct {
	AnimationState AnimationState
	CurrentIdx     uint16 // The current index in the frame, not the sheet
	CounterMs      uint64
}

type Animation struct {
	sheetAssetTag common.ImageAssetTag
	stateFrames   map[AnimationState][]AnimationFrame
	frameState    FrameState
	queuedState   *AnimationState
}

func NewAnimationComponent(
	sheetAssetTag common.ImageAssetTag,
	stateFrames map[AnimationState][]AnimationFrame,
) (*Animation, error) {
	return &Animation{
		sheetAssetTag: sheetAssetTag,
		stateFrames:   stateFrames,
	}, nil
}

func (x *Animation) State() AnimationState {
	return x.frameState.AnimationState
}

func (x *Animation) SetQueuedStateIfNone(newState AnimationState) error {
	if _, ok := x.stateFrames[newState]; !ok {
		return fmt.Errorf("animation state %d not found", newState)
	}

	if x.queuedState == nil {
		x.queuedState = &newState
	}

	return nil
}

// Restarts the animation in the new state, from the sprite's first frame
func (x *ECSContainer) SetAnimationState(e common.EntityId, newState AnimationState) error {
	a, ok := x.Animations.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no animation", e)
	}

	s, ok := x.Sprites.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no sprite to animate", e)
	}

	if _, ok := a.stateFrames[newState]; !ok {
		return fmt.Errorf("animation state %d not found for entity %d", newState, e)
	}

	a.frameState.AnimationState = newState
	a.frameState.CurrentIdx = 0
	a.frameState.CounterMs = 0
	s.subImageIdx = 0

	return nil
}

func (x *ECSContainer) TickAnimation(e common.EntityId) error {
	a, ok := x.Animations.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no animation", e)
	}

	s, ok := x.Sprites.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no sprite to animate", e)
	}

	currentState := a.frameState.AnimationState
	currentIdx := a.frameState.CurrentIdx

	frames, ok := a.stateFrames[currentState]
	if !ok {
		return fmt.Errorf("animation state %d not found for entity %d", currentState, e)
	}

	if int(currentIdx) >= len(frames) {
		return fmt.Errorf("current frame index %d out of bounds for animation state %d of entity %d", currentIdx, currentState, e)
	}

	frame := frames[currentIdx]

	a.frameState.CounterMs += data.TickMs

	if a.frameState.CounterMs >= frame.DurationMs {
		if a.queuedState != nil {
			a.frameState.AnimationState = *a.queuedState
			a.queuedState = nil
		} else {
			a.frameState.CounterMs = 0
			a.frameState.CurrentIdx = (a.frameState.CurrentIdx + 1) % uint16(len(frames))
			s.subImageIdx = int(a.stateFrames[currentState][currentIdx].FrameIdx)
		}
	}

	return nil
}

func (x Animation) Copy() Animation {
	sFrames := make(map[AnimationState][]AnimationFrame)
	for k, v := range x.stateFrames {
		sFrames[k] = make([]AnimationFrame, len(v))
		copy(sFrames[k], v)
	}

	var queuedState *AnimationState
	if x.queuedState != nil {
		qs := *x.queuedState
		queuedState = &qs
	}

	return Animation{
		sheetAssetTag: x.sheetAssetTag,
		stateFrames:   sFrames,
		frameState:    x.frameState,
		queuedState:   queuedState,
	}
}

func (x Animation) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Animations.addComponent(e, x.Copy())
}

type animationDto struct {
	SheetAssetTag common.ImageAssetTag
	StateFrames   map[AnimationState][]AnimationFrame
	FrameState    FrameState
	QueuedState   *AnimationState
}

func (x Animation) toDto() (ComponentDto, error) {
	x = x.Copy()

	return &animationDto{
		SheetAssetTag: x.sheetAssetTag,
		StateFrames:   x.stateFrames,
		FrameState:    x.frameState,
		QueuedState:   x.queuedState,
	}, nil
}

func (x *animationDto) toComponent() (Component, error) {
	return &Animation{
		sheetAssetTag: x.SheetAssetTag,
		stateFrames:   x.StateFrames,
		frameState:    x.FrameState,
		queuedState:   x.QueuedState,
	}, nil
}
