package components

import "ebittest/ecs/ecscommon"

// Do not instantiate directly, use NewParentComp().
type Parent struct {
	Entity ecscommon.EntityId
}

func (Parent) isComponent() {}
