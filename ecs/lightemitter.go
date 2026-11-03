package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

// Gives light from its holder's tile while it's held and enabled, like a torch in hand: level on that tile, fading to
// nothing rangeTiles away
type LightEmitter struct {
	level      uint8
	rangeTiles int
}

func NewLightEmitterComponent(level uint8, rangeTiles int) (*LightEmitter, error) {
	if level == 0 || rangeTiles <= 0 {
		return nil, fmt.Errorf("light %d up to %d tiles away isn't positive", level, rangeTiles)
	}
	return &LightEmitter{level: level, rangeTiles: rangeTiles}, nil
}

func (x LightEmitter) Copy() LightEmitter {
	return x
}

func (x LightEmitter) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.LightEmitters.addComponent(e, x.Copy())
}

type lightEmitterDto struct {
	Level      uint8
	RangeTiles int
}

func (x LightEmitter) toDto() (ComponentDto, error) {
	return &lightEmitterDto{Level: x.level, RangeTiles: x.rangeTiles}, nil
}

func (x lightEmitterDto) toComponent() (Component, error) {
	return NewLightEmitterComponent(x.Level, x.RangeTiles)
}
