package ecs

import (
	"ebittest/assetmanager"
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

type animation struct {
	sheetAssetTag assetmanager.ImageAssetTag
	stateFrames   map[AnimationState][]AnimationFrame
	frameState    FrameState
	queuedState   *AnimationState
}

func (animation) isComponent() {}

func (x animation) Copy() animation {
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

	return animation{
		sheetAssetTag: x.sheetAssetTag,
		stateFrames:   sFrames,
		frameState:    x.frameState,
		queuedState:   queuedState,
	}
}
