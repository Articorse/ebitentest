package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

type transformManager struct{}

func NewTransformComponent(pos utils.Vec2, scale float64, rotation float64) *transform {
	return &transform{pos: pos, scale: scale, rotation: rotation}
}

func (*transformManager) GetLocalPos(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	return traComp.pos, nil
}

func (*transformManager) GetWorldPos(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		return traComp.pos, nil
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
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}, nil
}

func (*transformManager) GetLocalRotation(
	e common.EntityId,
	world *World,
) (float64, error) {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	return traComp.rotation, nil
}

func (*transformManager) GetWorldRotation(
	e common.EntityId,
	world *World,
) (float64, error) {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		return traComp.rotation, nil
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	return pWorldRot + traComp.rotation, nil
}

func (*transformManager) GetLocalScale(
	e common.EntityId,
	world *World,
) float64 {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return 0
	}

	return traComp.scale
}

func (*transformManager) GetWorldScale(
	e common.EntityId,
	world *World,
) (float64, error) {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		return traComp.scale, nil
	}

	pWorldSca, err := tm.GetWorldScale(parEntity, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	return pWorldSca * traComp.scale, nil
}

func (*transformManager) SetLocalPos(
	e common.EntityId,
	pos utils.Vec2,
	world *World,
) error {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	traComp.pos = pos
	return nil
}

func (*transformManager) AddLocalPos(
	e common.EntityId,
	pos utils.Vec2,
	world *World,
) error {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	traComp.pos = traComp.pos.Add(pos)
	return nil
}

func (*transformManager) SetWorldPos(
	e common.EntityId,
	pos utils.Vec2,
	world *World,
) error {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		traComp.pos = pos
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

	traComp.pos = utils.Vec2{
		X: (traComp.pos.X*cos - traComp.pos.Y*sin) - pWorldPos.X,
		Y: (traComp.pos.X*sin + traComp.pos.Y*cos) - pWorldPos.Y,
	}

	return nil
}

func (*transformManager) SetLocalRotation(
	e common.EntityId,
	rot float64,
	world *World,
) error {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	traComp.rotation = rot
	return nil
}

func (*transformManager) AddLocalRotation(
	e common.EntityId,
	rot float64,
	world *World,
) error {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	traComp.rotation += rot
	return nil
}

func (*transformManager) SetWorldRotation(
	e common.EntityId,
	rot float64,
	world *World,
) error {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		traComp.rotation = rot
		return nil
	}

	pWorldRot, err := tm.GetWorldRotation(parEntity, world)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	traComp.rotation = rot - pWorldRot

	return nil
}

func (*transformManager) SetLocalScale(
	e common.EntityId,
	scale float64,
	world *World,
) error {
	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	traComp.scale = scale
	return nil
}

func (*transformManager) SetWorldScale(
	e common.EntityId,
	scale float64,
	world *World,
) error {
	pm := parentManager{}
	tm := transformManager{}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		traComp.scale = scale
		return nil
	}

	pWorldScale, err := tm.GetWorldScale(parEntity, world)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	traComp.scale = scale / pWorldScale

	return nil
}
