package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type FacePositionManager struct{}

func NewFacePositionComponent(pos utils.Vec2, enabled bool) *facePosition {
	return &facePosition{
		enabled: enabled,
		pos:     pos,
	}
}

func (*FacePositionManager) GetEnabled(e common.EntityId, world *World) (bool, error) {
	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	return fpComp.enabled, nil
}

func (*FacePositionManager) Enable(e common.EntityId, world *World) error {
	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.enabled = true

	return nil
}

func (*FacePositionManager) Disable(e common.EntityId, world *World) error {
	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.enabled = false

	return nil
}

func (*FacePositionManager) GetLocalPos(e common.EntityId, world *World) (utils.Vec2, error) {
	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	return fpComp.pos, nil
}

func (*FacePositionManager) GetWorldPos(e common.EntityId, world *World) (utils.Vec2, error) {
	tm := TransformManager{}
	pm := ParentManager{}

	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		return fpComp.pos, nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, world)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, world)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2{
		X: pWorldPos.X + (fpComp.pos.X*cos - fpComp.pos.Y*sin),
		Y: pWorldPos.Y + (fpComp.pos.X*sin + fpComp.pos.Y*cos),
	}, nil
}

func (*FacePositionManager) SetLocalPos(e common.EntityId, pos utils.Vec2, world *World) error {
	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.pos = pos

	return nil
}

func (*FacePositionManager) SetWorldPos(
	e common.EntityId,
	pos utils.Vec2,
	world *World,
) error {
	pm := ParentManager{}
	tm := TransformManager{}

	fpComp, err := world.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get facePosition of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		fpComp.pos = pos
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, world)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, world)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	fpComp.pos = utils.Vec2{
		X: (fpComp.pos.X*cos - fpComp.pos.Y*sin) - pWorldPos.X,
		Y: (fpComp.pos.X*sin + fpComp.pos.Y*cos) - pWorldPos.Y,
	}

	return nil
}
