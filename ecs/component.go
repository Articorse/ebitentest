package ecs

import "ebittest/ecs/common"

type Component interface {
	addTo(ecsContainer *ECSContainer, e common.EntityId)
	toDto() (ComponentDto, error)
}

type ComponentDto interface {
	toComponent() (Component, error)
}
