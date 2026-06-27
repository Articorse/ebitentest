package ecs

type Component interface {
	isComponent()
}

type ComponentDto interface {
	isComponentDto()
}
