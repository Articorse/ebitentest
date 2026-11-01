package ecs

import "ebittest/ecs/common"

type Persistent struct {
	// TODO: Add disable/despawn range
}

func NewPersistentComponent() *Persistent {
	return &Persistent{}
}

func (x Persistent) Copy() Persistent {
	return Persistent{}
}

func (x Persistent) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Persistents.addComponent(e, x.Copy())
}

type persistentDto struct {
}

func (x Persistent) toDto() (ComponentDto, error) {
	return &persistentDto{}, nil
}

func (x persistentDto) toComponent() (Component, error) {
	return &Persistent{}, nil
}
