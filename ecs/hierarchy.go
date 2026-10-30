package ecs

import (
	"ebittest/ecs/common"
	"slices"
)

type hierarchy struct {
	parent   common.EntityId
	children []common.EntityId
}

func (hierarchy) isComponent() {}

func (x hierarchy) Copy() hierarchy {
	return hierarchy{
		parent:   x.parent,
		children: slices.Clone(x.children),
	}
}

type hierarchyDto struct {
	Parent   common.EntityId
	Children []common.EntityId
}

func (hierarchyDto) isComponentDto() {}

func (x hierarchy) ToDto() hierarchyDto {
	return hierarchyDto{
		Parent:   x.parent,
		Children: slices.Clone(x.children),
	}
}

func (x hierarchyDto) ToComponent() *hierarchy {
	return &hierarchy{
		parent:   x.Parent,
		children: x.Children,
	}
}
