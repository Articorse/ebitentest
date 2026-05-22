package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"math"
)

type ParentManager struct{}

func NewParentComponent() *Parent {
	return &Parent{Entity: -1}
}

func (*ParentManager) Attach(
	c ecscommon.EntityId,
	p ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	tm := TransformManager{}

	parComp, ok := parents[c]
	if ok {
		if err := tm.Detach(c, transforms, parents); err != nil {
			return fmt.Errorf("error during detach: %v", err)
		}
	}

	parComp.Entity = p

	traComp, ok := transforms[c]
	if !ok {
		return fmt.Errorf("could not find transform of entity %d", c)
	}

	pTraComp := transforms[p]
	if pTraComp == nil {
		return fmt.Errorf("could not get parent while attaching entity %d to entity %d", c, p)
	}

	pWorldPos, err := tm.GetWorldPos(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.Entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.Entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parComp.Entity, err)
	}

	cos := math.Cos(pTraComp.rotation)
	sin := math.Sin(pTraComp.rotation)

	traComp.pos = utils.Vec2{
		X: (traComp.pos.X*cos - traComp.pos.Y*sin) - pWorldPos.X,
		Y: (traComp.pos.X*sin + traComp.pos.Y*cos) - pWorldPos.Y,
	}
	traComp.scale = traComp.scale / pWorldScale
	traComp.rotation = traComp.rotation - pWorldRot

	return nil
}

// If error handling is changed, check Attach()
func (*TransformManager) Detach(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	parComp, ok := parents[e]
	if !ok {
		return nil
	}

	if parComp.Entity == -1 {
		return nil
	}

	traComp, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	tm := TransformManager{}

	pWorldPos, err := tm.GetWorldPos(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.Entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.Entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.Entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parComp.Entity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2{
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}

	traComp.scale = pWorldScale
	traComp.rotation = pWorldRot

	parComp.Entity = -1

	return nil
}
