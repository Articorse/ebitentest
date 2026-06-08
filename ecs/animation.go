package ecs

import (
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

type AnimationState uint16

const (
	Anim_Idle AnimationState = iota
	Anim_Use
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
	sheet       *ebiten.Image
	frameSize   utils.Vec2
	stateFrames map[AnimationState][]AnimationFrame
	frameState  FrameState
	queuedState *AnimationState
}

func (animation) isComponent() {}

func (x animation) Copy() animation {
	return animation{
		sheet:       x.sheet,
		frameSize:   x.frameSize,
		stateFrames: x.stateFrames,
		frameState:  x.frameState,
		queuedState: x.queuedState,
	}
}
