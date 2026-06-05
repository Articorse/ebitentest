package components

import "ebittest/ecs/ecscommon"

// Do not instantiate directly, use NewParentComp().
type Parent struct {
	entity ecscommon.EntityId
}

func (Parent) isComponent() {}

func (x Parent) Copy() Parent {
	return Parent{
		entity: x.entity,
	}
}
