package ecs

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"slices"
)

type World struct {
	nextEntity ecscommon.EntityId
	Entities   []ecscommon.EntityId
	Inputs     map[ecscommon.EntityId]*components.Input
	Parents    map[ecscommon.EntityId]*components.Parent
	Children   map[ecscommon.EntityId]*components.Children
	Transforms map[ecscommon.EntityId]*components.Transform
	Velocities map[ecscommon.EntityId]*components.Velocity
	Sprites    map[ecscommon.EntityId]*components.Sprite
	Colliders  map[ecscommon.EntityId]*components.Collider
}

func NewWorld() *World {
	return &World{
		nextEntity: 0,
		Entities:   []ecscommon.EntityId{},
		Inputs:     make(map[ecscommon.EntityId]*components.Input),
		Parents:    make(map[ecscommon.EntityId]*components.Parent),
		Children:   make(map[ecscommon.EntityId]*components.Children),
		Transforms: make(map[ecscommon.EntityId]*components.Transform),
		Velocities: make(map[ecscommon.EntityId]*components.Velocity),
		Sprites:    make(map[ecscommon.EntityId]*components.Sprite),
		Colliders:  make(map[ecscommon.EntityId]*components.Collider),
	}
}

func (x *World) AddEntity() ecscommon.EntityId {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *World) RemoveEntity(e ecscommon.EntityId) error {
	x.Entities = slices.DeleteFunc(x.Entities,
		func(ent ecscommon.EntityId) bool { return ent == e })

	delete(x.Parents, e)
	delete(x.Children, e)
	delete(x.Transforms, e)
	delete(x.Velocities, e)
	delete(x.Sprites, e)
	delete(x.Colliders, e)

	for _, p := range x.Parents {
		if p.Entity == e {
			p.Entity = -1
		}
	}

	for _, c := range x.Children {
		if slices.Contains(c.Entities, &e) {
			slices.DeleteFunc(c.Entities, func(ent *ecscommon.EntityId) bool { return *ent == e })
		}
	}

	return nil
}
