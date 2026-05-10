package ecs

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"maps"
	"slices"
)

type World struct {
	nextEntity ecscommon.Entity
	Entities   []ecscommon.Entity
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

func (x *World) RemoveEntity(e ecscommon.Entity) error {
	x.Entities = slices.DeleteFunc(x.Entities,
		func(ent ecscommon.Entity) bool { return ent == e })

	maps.DeleteFunc(x.Parents,
		func(k ecscommon.Entity, _ *components.Parent) bool { return k == e })

	maps.DeleteFunc(x.Children,
		func(k ecscommon.Entity, _ *components.Children) bool { return k == e })

	maps.DeleteFunc(x.Transforms,
		func(k ecscommon.Entity, _ *components.Transform) bool { return k == e })

	maps.DeleteFunc(x.Velocities,
		func(k ecscommon.Entity, _ *components.Velocity) bool { return k == e })

	maps.DeleteFunc(x.Sprites,
		func(k ecscommon.Entity, _ *components.Sprite) bool { return k == e })

	maps.DeleteFunc(x.Colliders,
		func(k ecscommon.Entity, _ *components.Collider) bool { return k == e })

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
