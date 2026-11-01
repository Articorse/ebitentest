package ecs

import (
	"ebittest/ecs/common"
	"slices"
)

// Decides what happens to a child when its parent is stored, removed or killed.
// Every kind moves with its parent the same way.
type LinkKindEnum uint8

const (
	// Stored with the parent if persistent, removed with it
	Link_Part LinkKindEnum = iota
	// Always stored with the parent, dropped when it's killed
	Link_Held
	// Never stored with the parent, detached when it's stored or removed
	Link_Attached
)

type hierarchy struct {
	parent   common.EntityId
	linkKind LinkKindEnum
	children []common.EntityId
}

func (x hierarchy) Copy() hierarchy {
	return hierarchy{
		parent:   x.parent,
		linkKind: x.linkKind,
		children: slices.Clone(x.children),
	}
}

func (x hierarchy) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Hierarchies.addComponent(e, x.Copy())
}

type hierarchyDto struct {
	Parent   common.EntityId
	LinkKind LinkKindEnum
	Children []common.EntityId
}

func (x hierarchy) toDto() (ComponentDto, error) {
	return &hierarchyDto{
		Parent:   x.parent,
		LinkKind: x.linkKind,
		Children: slices.Clone(x.children),
	}, nil
}

func (x hierarchyDto) toComponent() (Component, error) {
	return &hierarchy{
		parent:   x.Parent,
		linkKind: x.LinkKind,
		children: x.Children,
	}, nil
}
