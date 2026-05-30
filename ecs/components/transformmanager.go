package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"math"
)

type TransformManager struct{}

func NewTransformComponent(pos utils.Vec2, scale float64, rotation float64) *Transform {
	return &Transform{pos: pos, prevPos: pos, scale: scale, rotation: rotation}
}

func (*TransformManager) GetLocalPos(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
) (utils.Vec2, error) {
	traComp, ok := transforms[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	return traComp.pos, nil
}

func (*TransformManager) GetWorldPos(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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

func (*TransformManager) GetLocalPrevPos(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
) (utils.Vec2, error) {
	traComp, ok := transforms[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	return traComp.prevPos, nil
}

func (*TransformManager) GetWorldPrevPos(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (utils.Vec2, error) {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return traComp.prevPos, nil
	}

	pWorldPos, err := tm.GetWorldPrevPos(parEntity, transforms, parents)
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
		X: pWorldPos.X + (traComp.prevPos.X*cos - traComp.prevPos.Y*sin),
		Y: pWorldPos.Y + (traComp.prevPos.X*sin + traComp.prevPos.Y*cos),
	}, nil
}

func (*TransformManager) GetLocalRotation(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
) (float64, error) {
	traComp, ok := transforms[e]
	if !ok {
		return 0, fmt.Errorf("could not get transform of entity %d", e)
	}

	return traComp.rotation, nil
}

func (*TransformManager) GetWorldRotation(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
) float64 {
	traComp, ok := transforms[e]
	if !ok {
		return 0
	}

	return traComp.scale
}

func (*TransformManager) GetWorldScale(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
	e ecscommon.EntityId,
	pos utils.Vec2,
	transforms map[ecscommon.EntityId]*Transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.pos = pos
	return nil
}

func (*TransformManager) SetWorldPos(
	e ecscommon.EntityId,
	pos utils.Vec2,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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

func (*TransformManager) SetLocalPrevPos(
	e ecscommon.EntityId,
	prevPos utils.Vec2,
	transforms map[ecscommon.EntityId]*Transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.prevPos = prevPos
	return nil
}

func (*TransformManager) SetWorldPrevPos(
	e ecscommon.EntityId,
	pos utils.Vec2,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	pm := ParentManager{}
	tm := TransformManager{}

	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		traComp.prevPos = pos
		return nil
	}

	pWorldPrevPos, err := tm.GetWorldPrevPos(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parEntity, err)
	}

	// TODO: Check if I might not need to add a PrevRotation to Transform
	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2{
		X: (traComp.pos.X*cos - traComp.pos.Y*sin) - pWorldPrevPos.X,
		Y: (traComp.pos.X*sin + traComp.pos.Y*cos) - pWorldPrevPos.Y,
	}

	return nil
}

func (*TransformManager) SetLocalRotation(
	e ecscommon.EntityId,
	rot float64,
	transforms map[ecscommon.EntityId]*Transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.rotation = rot
	return nil
}

func (*TransformManager) SetWorldRotation(
	e ecscommon.EntityId,
	rot float64,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
	e ecscommon.EntityId,
	scale float64,
	transforms map[ecscommon.EntityId]*Transform,
) error {
	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	traComp.scale = scale
	return nil
}

func (*TransformManager) SetWorldScale(
	e ecscommon.EntityId,
	scale float64,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
