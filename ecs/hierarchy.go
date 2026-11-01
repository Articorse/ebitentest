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

// Links are only changed through ECSContainer.Attach and Detach, which keep both ends and the transforms in step
type Hierarchy struct {
	parent   common.EntityId
	linkKind LinkKindEnum
	children []common.EntityId
}

func NewHierarchyComponent() *Hierarchy {
	return &Hierarchy{parent: -1}
}

func (x *Hierarchy) Parent() common.EntityId {
	return x.parent
}

func (x *Hierarchy) LinkKind() LinkKindEnum {
	return x.linkKind
}

// A copy, since the children change as entities are attached and detached while callers still hold the result
func (x *Hierarchy) Children() []common.EntityId {
	return slices.Clone(x.children)
}

func (x Hierarchy) Copy() Hierarchy {
	return Hierarchy{
		parent:   x.parent,
		linkKind: x.linkKind,
		children: slices.Clone(x.children),
	}
}

func (x Hierarchy) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Hierarchies.addComponent(e, x.Copy())
}

type hierarchyDto struct {
	Parent   common.EntityId
	LinkKind LinkKindEnum
	Children []common.EntityId
}

func (x Hierarchy) toDto() (ComponentDto, error) {
	return &hierarchyDto{
		Parent:   x.parent,
		LinkKind: x.linkKind,
		Children: slices.Clone(x.children),
	}, nil
}

func (x hierarchyDto) toComponent() (Component, error) {
	return &Hierarchy{
		parent:   x.Parent,
		linkKind: x.LinkKind,
		children: x.Children,
	}, nil
}
