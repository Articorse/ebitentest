package ecs

import "ebittest/ecs/common"

// Keeps an entity in memory, saved and streamed as usual, but out of the simulation and out of sight. Systems skip
// it, while everything that only stores entities, such as saving and chunk streaming, doesn't care.
type Disabled struct{}

func NewDisabledComponent() *Disabled {
	return &Disabled{}
}

func (x *ECSContainer) IsDisabled(e common.EntityId) bool {
	return x.Disabled.HasComponent(e)
}

// Disables or enables e along with everything attached to it, so that nothing attached to it keeps simulating
func (x *ECSContainer) SetDisabled(e common.EntityId, disabled bool) {
	if disabled {
		x.Disabled.addComponent(e, Disabled{})
	} else {
		x.Disabled.deleteEntity(e)
	}

	for _, c := range x.Children(e) {
		x.SetDisabled(c, disabled)
	}
}

func (x Disabled) Copy() Disabled {
	return Disabled{}
}

func (x Disabled) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Disabled.addComponent(e, x.Copy())
}

type disabledDto struct{}

func (x Disabled) toDto() (ComponentDto, error) {
	return &disabledDto{}, nil
}

func (x disabledDto) toComponent() (Component, error) {
	return NewDisabledComponent(), nil
}
