package ecs

import "ebittest/ecs/common"

// Lets an entity be put in an inventory. Each item is its own entity, so two items of a kind can differ in any of
// their components.
type Item struct{}

func NewItemComponent() *Item {
	return &Item{}
}

func (x Item) Copy() Item {
	return Item{}
}

func (x Item) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Items.addComponent(e, x.Copy())
}

type itemDto struct{}

func (x Item) toDto() (ComponentDto, error) {
	return &itemDto{}, nil
}

func (x itemDto) toComponent() (Component, error) {
	return NewItemComponent(), nil
}
