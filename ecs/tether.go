package ecs

import "ebittest/ecs/common"

type tether struct {
	anchor common.EntityId
}

func (x tether) Copy() tether {
	return tether{
		anchor: x.anchor,
	}
}

func (x tether) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Tethers.addComponent(e, x.Copy())
}

type tetherDto struct {
	Anchor common.EntityId
}

func (x tether) toDto() (ComponentDto, error) {
	return &tetherDto{
		Anchor: x.anchor,
	}, nil
}

func (x tetherDto) toComponent() (Component, error) {
	return &tether{
		anchor: x.Anchor,
	}, nil
}
