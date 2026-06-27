package ecs

import "ebittest/ecs/common"

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
	sheetAssetTag common.ImageAssetTag
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

type animationDto struct {
	SheetAssetTag common.ImageAssetTag
	StateFrames   map[AnimationState][]AnimationFrame
	FrameState    FrameState
	QueuedState   *AnimationState
}

func (animationDto) isComponentDto() {}

func (x animation) ToDto() animationDto {
	return animationDto{
		SheetAssetTag: x.sheetAssetTag,
		StateFrames:   x.stateFrames,
		FrameState:    x.frameState,
		QueuedState:   x.queuedState,
	}
}

func (x *animationDto) ToComponent() *animation {
	return &animation{
		sheetAssetTag: x.SheetAssetTag,
		stateFrames:   x.StateFrames,
		frameState:    x.FrameState,
		queuedState:   x.QueuedState,
	}
}
