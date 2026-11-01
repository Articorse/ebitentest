package ecs

import "ebittest/ecs/common"

type tether struct {
	anchor common.EntityId
}

func (tether) isComponent() {}

func (x tether) Copy() tether {
	return tether{
		anchor: x.anchor,
	}
}

type tetherDto struct {
	Anchor common.EntityId
}

func (tetherDto) isComponentDto() {}

func (x tether) ToDto() tetherDto {
	return tetherDto{
		Anchor: x.anchor,
	}
}

func (x tetherDto) ToComponent() *tether {
	return &tether{
		anchor: x.Anchor,
	}
}
