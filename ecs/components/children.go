package components

import "ebittest/ecs/ecscommon"

// Do not instantiate directly, use NewChildrenComp().
type Children struct {
	Entities []*ecscommon.EntityId
}

func NewChildrenComponent() *Children {
	return &Children{}
}
