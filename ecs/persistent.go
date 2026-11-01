package ecs

import "ebittest/ecs/common"

type persistent struct {
	// TODO: Add disable/despawn range
}

func (x persistent) Copy() persistent {
	return persistent{}
}

func (x persistent) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Persistents.addComponent(e, x.Copy())
}

type persistentDto struct {
}

func (x persistent) toDto() (ComponentDto, error) {
	return &persistentDto{}, nil
}

func (x persistentDto) toComponent() (Component, error) {
	return &persistent{}, nil
}
