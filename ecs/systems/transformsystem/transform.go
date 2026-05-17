package transformsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"math"
)

// TODO: Calculate transforms on each tick to avoid recalculating for each child

func GetTransform(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (*components.Transform, error) {
	pt, err := GetParentTransform(e, transforms, parents)
	if err != nil {
		return nil, fmt.Errorf("could not get parent transform for entity %d", e)
	}

	t, ok := transforms[e]
	if !ok {
		return nil, fmt.Errorf("could not find transform of entity %d", e)
	}

	cos := math.Cos(pt.GetRotation())
	sin := math.Sin(pt.GetRotation())

	return components.NewTransformComponent(
		utils.Vec2{
			X: pt.GetPos().X + (t.GetPos().X*cos - t.GetPos().Y*sin),
			Y: pt.GetPos().Y + (t.GetPos().X*sin + t.GetPos().Y*cos)},
		t.GetScale()*pt.GetScale(),
		t.GetRotation()+pt.GetRotation(),
	), nil
}

func Attach(
	c ecscommon.EntityId,
	p ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	if err := Detach(c, transforms, parents); err != nil {
		return fmt.Errorf("error during detach", err)
	}

	pt, err := GetParentTransform(c, transforms, parents)
	if err != nil {
		return fmt.Errorf("could not get parent while attaching entity %d to entity %d", c, p)
	}

	pc, err := GetParentComponent(c, parents)
	if err != nil {
		return fmt.Errorf("could not get parent component of entity %d", c)
	}

	t, ok := transforms[c]
	if !ok {
		return fmt.Errorf("could not find transform of entity %d", c)
	}

	cos := math.Cos(pt.GetRotation())
	sin := math.Sin(pt.GetRotation())

	t = components.NewTransformComponent(
		utils.Vec2{
			X: pt.GetPos().X + (t.GetPos().X*cos - t.GetPos().Y*sin),
			Y: pt.GetPos().Y + (t.GetPos().X*sin + t.GetPos().Y*cos)},
		t.GetScale()/pt.GetScale(),
		t.GetRotation()-pt.GetRotation(),
	)

	pc.Entity = &p

	return nil
}

// If error handling is changed, check Attach()
func Detach(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	pt, err := GetParentTransform(e, transforms, parents)
	if err != nil {
		return fmt.Errorf("could not get parent transform of entity %d", e)
	}

	pc, err := GetParentComponent(e, parents)
	if err != nil {
		return fmt.Errorf("could not get parent component of entity %d", e)
	}

	t, ok := transforms[e]
	if !ok {
		return fmt.Errorf("could not get transform of entity %d", e)
	}

	cos := math.Cos(pt.GetRotation())
	sin := math.Sin(pt.GetRotation())

	t = components.NewTransformComponent(
		utils.Vec2{
			X: pt.GetPos().X + (t.GetPos().X*cos - t.GetPos().Y*sin),
			Y: pt.GetPos().Y + (t.GetPos().X*sin + t.GetPos().Y*cos)},
		t.GetScale()*pt.GetScale(),
		t.GetRotation()+pt.GetRotation(),
	)

	pc.Entity = nil

	return nil
}

func GetParentTransform(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) (*components.Transform, error) {

	pc, err := GetParentComponent(e, parents)
	if err != nil {
		return nil, fmt.Errorf("could not get parent component of entity %d", e)
	}

	pt, ok := transforms[*pc.Entity]
	if !ok {
		return nil, fmt.Errorf("could not get parent transform of entity %d", e)
	}

	return pt, nil
}

func GetParentComponent(
	e ecscommon.EntityId,
	parents map[ecscommon.EntityId]*components.Parent,
) (*components.Parent, error) {
	pc, ok := parents[e]
	if !ok {
		return nil, fmt.Errorf("tried to detach with no parent component for entity %d", e)
	}

	return pc, nil
}
