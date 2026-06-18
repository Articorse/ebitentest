package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type animationManager struct{}

func NewAnimationComponent(
	sheetUri string,
	frameSize utils.Vec2,
	stateFrames map[AnimationState][]AnimationFrame,
) (*animation, error) {
	sheetImg, _, err := ebitenutil.NewImageFromFile(sheetUri)
	if err != nil {
		return nil, fmt.Errorf("failed to load sprite image: %w", err)
	}

	return &animation{
		sheet:       sheetImg,
		frameSize:   frameSize,
		stateFrames: stateFrames,
	}, nil
}

func NewAnimationComponentWithSheet(
	sheet *ebiten.Image,
	frameSize utils.Vec2,
	stateFrames map[AnimationState][]AnimationFrame,
) (*animation, error) {
	return &animation{
		sheet:       sheet,
		frameSize:   frameSize,
		stateFrames: stateFrames,
	}, nil
}

func (animationManager) GetState(
	e common.EntityId,
	world *World,
) (AnimationState, error) {
	animComp, err := world.Animations.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	return animComp.frameState.AnimationState, nil
}

func (animationManager) GetCurrentFrame(
	e common.EntityId,
	world *World,
) (*ebiten.Image, error) {
	animComp, err := world.Animations.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	currentState := animComp.frameState.AnimationState
	currentIdx := animComp.frameState.CurrentIdx

	frames, ok := animComp.stateFrames[currentState]
	if !ok {
		return nil, fmt.Errorf("animation state %d not found for entity %d", currentState, e)
	}

	if int(currentIdx) >= len(frames) {
		return nil, fmt.Errorf("current frame index %d out of bounds for animation state %d of entity %d", currentIdx, currentState, e)
	}

	frame := frames[currentIdx]

	sx := (frame.FrameIdx * uint16(animComp.frameSize.X)) % uint16(animComp.sheet.Bounds().Dx())
	sy := ((frame.FrameIdx * uint16(animComp.frameSize.X)) / uint16(animComp.sheet.Bounds().Dx())) * uint16(animComp.frameSize.Y)

	subImage := animComp.sheet.SubImage(image.Rect(int(sx), int(sy), int(sx)+int(animComp.frameSize.X), int(sy)+int(animComp.frameSize.Y))).(*ebiten.Image)

	return subImage, nil
}

func (animationManager) SetState(
	e common.EntityId,
	newState AnimationState,
	world *World,
) error {
	animComp, err := world.Animations.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	if _, ok := animComp.stateFrames[newState]; !ok {
		return fmt.Errorf("animation state %d not found for entity %d", newState, e)
	}

	animComp.frameState.AnimationState = newState
	animComp.frameState.CurrentIdx = 0
	animComp.frameState.CounterMs = 0

	return nil
}

func (animationManager) SetQueuedStateIfNone(
	e common.EntityId,
	newState AnimationState,
	world *World,
) error {
	animComp, err := world.Animations.getComponent(e)
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
	world *World,
) error {
	animComp, err := world.Animations.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get animation component of entity %d: %v", e, err)
	}

	currentState := animComp.frameState.AnimationState
	currentIdx := animComp.frameState.CurrentIdx

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
		}
	}

	return nil
}
