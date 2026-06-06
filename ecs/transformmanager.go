package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type TransformManager struct{}

func NewTransformComponent(pos utils.Vec2, scale float64, rotation float64) *transform {
	return &transform{pos: pos, scale: scale, rotation: rotation}
}

func (*TransformManager) GetLocalPos(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
) (utils.Vec2, error) {
	traComp, ok := transforms[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	return traComp.pos, nil
}

func (*TransformManager) GetWorldPos(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) (utils.Vec2, error) {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return traComp.pos, nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, ok)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2{
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}, nil
}

func (*TransformManager) GetLocalRotation(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
) (float64, error) {
	traComp, ok := transforms[e]
	if !ok {
		return 0, fmt.Errorf("could not get transform of entity %d", e)
	}

	return traComp.rotation, nil
}

func (*TransformManager) GetWorldRotation(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) (float64, error) {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return 0, fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return traComp.rotation, nil
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	return pWorldRot + traComp.rotation, nil
}

func (*TransformManager) GetLocalScale(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
) float64 {
	traComp, ok := transforms[e]
	if !ok {
		return 0
	}

	return traComp.scale
}

func (*TransformManager) GetWorldScale(
	e common.EntityId,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) (float64, error) {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return 0, fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return traComp.scale, nil
	}

	pWorldSca, err := tm.GetWorldScale(parEntity, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	return pWorldSca * traComp.scale, nil
}

func (*TransformManager) SetLocalPos(
	e common.EntityId,
	pos utils.Vec2,
	transforms map[common.EntityId]*transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.pos = pos
	return nil
}

func (*TransformManager) SetWorldPos(
	e common.EntityId,
	pos utils.Vec2,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) error {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		traComp.pos = pos
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2{
		X: (traComp.pos.X*cos - traComp.pos.Y*sin) - pWorldPos.X,
		Y: (traComp.pos.X*sin + traComp.pos.Y*cos) - pWorldPos.Y,
	}

	return nil
}

func (*TransformManager) SetLocalRotation(
	e common.EntityId,
	rot float64,
	transforms map[common.EntityId]*transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.rotation = rot
	return nil
}

func (*TransformManager) SetWorldRotation(
	e common.EntityId,
	rot float64,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) error {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		traComp.rotation = rot
		return nil
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	traComp.rotation = pWorldRot + rot

	return nil
}

func (*TransformManager) SetLocalScale(
	e common.EntityId,
	scale float64,
	transforms map[common.EntityId]*transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.scale = scale
	return nil
}

func (*TransformManager) SetWorldScale(
	e common.EntityId,
	scale float64,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) error {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		traComp.scale = scale
		return nil
	}

	pWorldScale, err := tm.GetWorldScale(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	traComp.scale = scale / pWorldScale

	return nil
}
