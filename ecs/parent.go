package ecs

import "ebittest/ecs/common"

// Do not instantiate directly, use NewParentComp().
type parent struct {
	entity common.EntityId
}

func (parent) isComponent() {}

func (x parent) Copy() parent {
	return parent{
		entity: x.entity,
	}
}
