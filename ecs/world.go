package ecs

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"fmt"
	"slices"
)

type World struct {
	nextEntity ecscommon.Entity
	Entities   []ecscommon.Entity
	Players    map[ecscommon.PlayerId]*ecscommon.PlayerConfig
	Parents    map[ecscommon.Entity]*components.Parent
	Children   map[ecscommon.Entity]*components.Children
	Transforms map[ecscommon.Entity]*components.Transform
	Velocities map[ecscommon.Entity]*components.Velocity
	Sprites    map[ecscommon.Entity]*components.Sprite
	Colliders  map[ecscommon.Entity]*components.Collider
}

func NewWorld() *World {
	return &World{
		nextEntity: 0,
		Entities:   []ecscommon.Entity{},
		Players:    make(map[ecscommon.PlayerId]*ecscommon.PlayerConfig),
		Parents:    make(map[ecscommon.Entity]*components.Parent),
		Children:   make(map[ecscommon.Entity]*components.Children),
		Transforms: make(map[ecscommon.Entity]*components.Transform),
		Velocities: make(map[ecscommon.Entity]*components.Velocity),
		Sprites:    make(map[ecscommon.Entity]*components.Sprite),
		Colliders:  make(map[ecscommon.Entity]*components.Collider),
	}
}

func (x *World) AddEntity() ecscommon.Entity {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *World) AddPlayer(pId ecscommon.PlayerId, e ecscommon.Entity, km ecscommon.KeyMaps) (*ecscommon.PlayerConfig, error) {
	if _, ok := x.Players[pId]; ok {
		return nil, fmt.Errorf("player %s already exists", pId)
	}

	p := &ecscommon.PlayerConfig{Entity: e, KeyMaps: km}
	x.Players[pId] = p

	return p, nil
}

func (x *World) RemoveEntity(e ecscommon.Entity) error {
	x.Entities = slices.DeleteFunc(x.Entities,
		func(ent ecscommon.Entity) bool { return ent == e })

	delete(x.Parents, e)
	delete(x.Children, e)
	delete(x.Transforms, e)
	delete(x.Velocities, e)
	delete(x.Sprites, e)
	delete(x.Colliders, e)

	for _, p := range x.Parents {
		if *p.Entity == e {
			p.Entity = nil
		}
	}

	for _, c := range x.Children {
		if slices.Contains(c.Entities, &e) {
			slices.DeleteFunc(c.Entities, func(ent *ecscommon.Entity) bool { return *ent == e })
		}
	}

	return nil
}

func (x *World) RemovePlayer(pId ecscommon.PlayerId) error {
	if _, ok := x.Players[pId]; !ok {
		return fmt.Errorf("player %s not found", pId)
	}

	delete(x.Players, pId)
	return nil
}
