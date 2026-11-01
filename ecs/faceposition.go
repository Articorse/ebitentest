package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

// A position, relative to the parent if there is one, that the entity turns to face
type FacePosition struct {
	enabled bool
	pos     utils.Vec2f
}

func NewFacePositionComponent(pos utils.Vec2f, enabled bool) *FacePosition {
	return &FacePosition{
		enabled: enabled,
		pos:     pos,
	}
}

func (x *FacePosition) Enabled() bool {
	return x.enabled
}

func (x *FacePosition) Enable() {
	x.enabled = true
}

func (x *FacePosition) Disable() {
	x.enabled = false
}

func (x *FacePosition) LocalPos() utils.Vec2f {
	return x.pos
}

func (x *FacePosition) SetLocalPos(pos utils.Vec2f) {
	x.pos = pos
}

func (x *ECSContainer) FacePositionWorldPos(e common.EntityId) (utils.Vec2f, error) {
	fp, ok := x.FacePositions.Get(e)
	if !ok {
		return utils.Vec2f{}, fmt.Errorf("entity %d has no face position", e)
	}

	p := x.Parent(e)
	if p == -1 {
		return fp.pos, nil
	}

	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2f{
		X: pWorldPos.X + (fp.pos.X*cos - fp.pos.Y*sin),
		Y: pWorldPos.Y + (fp.pos.X*sin + fp.pos.Y*cos),
	}, nil
}

func (x *ECSContainer) SetFacePositionWorldPos(e common.EntityId, pos utils.Vec2f) error {
	fp, ok := x.FacePositions.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no face position", e)
	}

	p := x.Parent(e)
	if p == -1 {
		fp.pos = pos
		return nil
	}

	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(-pWorldRot)
	sin := math.Sin(-pWorldRot)
	relPos := pos.Subtract(pWorldPos)

	fp.pos = utils.Vec2f{
		X: relPos.X*cos - relPos.Y*sin,
		Y: relPos.X*sin + relPos.Y*cos,
	}

	return nil
}

func (x FacePosition) Copy() FacePosition {
	return FacePosition{
		enabled: x.enabled,
		pos:     x.pos,
	}
}

func (x FacePosition) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.FacePositions.addComponent(e, x.Copy())
}

type facePositionDto struct {
	Enabled bool
	Pos     utils.Vec2f
}

func (x FacePosition) toDto() (ComponentDto, error) {
	return &facePositionDto{
		Enabled: x.enabled,
		Pos:     x.pos,
	}, nil
}

func (x facePositionDto) toComponent() (Component, error) {
	return &FacePosition{
		enabled: x.Enabled,
		pos:     x.Pos,
	}, nil
}
