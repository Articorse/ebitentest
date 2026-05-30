package ecs

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"fmt"
	"slices"
)

type World struct {
	nextEntity ecscommon.EntityId
	Entities   []ecscommon.EntityId
	Inputs     map[ecscommon.EntityId]*components.Input
	Parents    map[ecscommon.EntityId]*components.Parent
	Transforms map[ecscommon.EntityId]*components.Transform
	Velocities map[ecscommon.EntityId]*components.Velocity
	Sprites    map[ecscommon.EntityId]*components.Sprite
	Colliders  map[ecscommon.EntityId]*components.Collider
	Platforms  map[ecscommon.EntityId]*components.Platform
}

func NewWorld() *World {
	return &World{
		nextEntity: 0,
		Entities:   []ecscommon.EntityId{},
		Inputs:     make(map[ecscommon.EntityId]*components.Input),
		Parents:    make(map[ecscommon.EntityId]*components.Parent),
		Transforms: make(map[ecscommon.EntityId]*components.Transform),
		Velocities: make(map[ecscommon.EntityId]*components.Velocity),
		Sprites:    make(map[ecscommon.EntityId]*components.Sprite),
		Colliders:  make(map[ecscommon.EntityId]*components.Collider),
		Platforms:  make(map[ecscommon.EntityId]*components.Platform),
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
	delete(x.Transforms, e)
	delete(x.Velocities, e)
	delete(x.Sprites, e)
	delete(x.Colliders, e)

	pm := components.ParentManager{}
	err := pm.RemoveParentFromAllEntities(e, x.Parents, x.Transforms)
	if err != nil {
		return fmt.Errorf("error removing entity %d from parent component of all entities: %v", e, err)
	}

	return nil
}
