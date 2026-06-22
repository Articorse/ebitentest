package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type facePositionManager struct{}

func NewFacePositionComponent(pos utils.Vec2, enabled bool) *facePosition {
	return &facePosition{
		enabled: enabled,
		pos:     pos,
	}
}

func (*facePositionManager) GetEnabled(e common.EntityId, ecsContainer *ECSContainer) (bool, error) {
	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	return fpComp.enabled, nil
}

func (*facePositionManager) Enable(e common.EntityId, ecsContainer *ECSContainer) error {
	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.enabled = true

	return nil
}

func (*facePositionManager) Disable(e common.EntityId, ecsContainer *ECSContainer) error {
	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.enabled = false

	return nil
}

func (*facePositionManager) GetLocalPos(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	return fpComp.pos, nil
}

func (*facePositionManager) GetWorldPos(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	tm := transformManager{}
	pm := parentManager{}

	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, ecsContainer)
	if parEntity == -1 {
		return fpComp.pos, nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, ecsContainer)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, ecsContainer)
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

func (*facePositionManager) SetLocalPos(e common.EntityId, pos utils.Vec2, ecsContainer *ECSContainer) error {
	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get face position component of entity %d: %v", e, err)
	}

	fpComp.pos = pos

	return nil
}

func (*facePositionManager) SetWorldPos(
	e common.EntityId,
	pos utils.Vec2,
	ecsContainer *ECSContainer,
) error {
	pm := parentManager{}
	tm := transformManager{}

	fpComp, err := ecsContainer.FacePositions.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get facePosition of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, ecsContainer)
	if parEntity == -1 {
		fpComp.pos = pos
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, ecsContainer)
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
