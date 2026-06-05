package ecs

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"fmt"
)

type World struct {
	nextEntity        ecscommon.EntityId
	Inputs            map[ecscommon.EntityId]*components.Input
	Parents           map[ecscommon.EntityId]*components.Parent
	Transforms        map[ecscommon.EntityId]*components.Transform
	Velocities        map[ecscommon.EntityId]*components.Velocity
	Sprites           map[ecscommon.EntityId]*components.Sprite
	CollisionLayers   map[ecscommon.EntityId]*components.CollisionLayer
	PhysicsColliders  map[ecscommon.EntityId]*components.PhysicsCollider
	PlatformColliders map[ecscommon.EntityId]*components.PlatformCollider
	HitboxColliders   map[ecscommon.EntityId]*components.HitboxCollider
	HurtboxColliders  map[ecscommon.EntityId]*components.HurtboxCollider
	Platforms         map[ecscommon.EntityId]*components.Platform
	Spawners          map[ecscommon.EntityId]*components.Spawner
	TimedLives        map[ecscommon.EntityId]*components.TimedLife
	Hitpoints         map[ecscommon.EntityId]*components.Hitpoints
	ContactDamages    map[ecscommon.EntityId]*components.ContactDamage
}

func NewWorld() *World {
	return &World{
		nextEntity:        0,
		Inputs:            make(map[ecscommon.EntityId]*components.Input),
		Parents:           make(map[ecscommon.EntityId]*components.Parent),
		Transforms:        make(map[ecscommon.EntityId]*components.Transform),
		Velocities:        make(map[ecscommon.EntityId]*components.Velocity),
		Sprites:           make(map[ecscommon.EntityId]*components.Sprite),
		CollisionLayers:   make(map[ecscommon.EntityId]*components.CollisionLayer),
		PhysicsColliders:  make(map[ecscommon.EntityId]*components.PhysicsCollider),
		PlatformColliders: make(map[ecscommon.EntityId]*components.PlatformCollider),
		HitboxColliders:   make(map[ecscommon.EntityId]*components.HitboxCollider),
		HurtboxColliders:  make(map[ecscommon.EntityId]*components.HurtboxCollider),
		Platforms:         make(map[ecscommon.EntityId]*components.Platform),
		Spawners:          make(map[ecscommon.EntityId]*components.Spawner),
		TimedLives:        make(map[ecscommon.EntityId]*components.TimedLife),
		Hitpoints:         make(map[ecscommon.EntityId]*components.Hitpoints),
		ContactDamages:    make(map[ecscommon.EntityId]*components.ContactDamage),
	}
}

func (x *World) AddEntity() ecscommon.EntityId {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *World) RemoveEntity(e ecscommon.EntityId) error {
	delete(x.Parents, e)
	delete(x.Transforms, e)
	delete(x.Velocities, e)
	delete(x.Sprites, e)
	delete(x.CollisionLayers, e)
	delete(x.PhysicsColliders, e)
	delete(x.PlatformColliders, e)
	delete(x.HitboxColliders, e)
	delete(x.HurtboxColliders, e)
	delete(x.Platforms, e)
	delete(x.Spawners, e)
	delete(x.TimedLives, e)
	delete(x.Hitpoints, e)
	delete(x.ContactDamages, e)

	pm := components.ParentManager{}
	err := pm.RemoveParentFromAllEntities(e, x.Parents, x.Transforms)
	if err != nil {
		return fmt.Errorf("error removing entity %d from parent component of all entities: %v", e, err)
	}

	return nil
}

func (x *World) AddComponent(e ecscommon.EntityId, comp components.Component) {
	switch c := comp.(type) {
	case *components.PhysicsCollider:
		col := c.Copy()
		x.PhysicsColliders[e] = &col
	case *components.CollisionLayer:
		cl := c.Copy()
		x.CollisionLayers[e] = &cl
	case *components.HitboxCollider:
		hb := c.Copy()
		x.HitboxColliders[e] = &hb
	case *components.HurtboxCollider:
		hb := c.Copy()
		x.HurtboxColliders[e] = &hb
	case *components.PlatformCollider:
		pc := c.Copy()
		x.PlatformColliders[e] = &pc
	case *components.Input:
		inp := c.Copy()
		x.Inputs[e] = &inp
	case *components.Parent:
		par := c.Copy()
		x.Parents[e] = &par
	case *components.Platform:
		plat := c.Copy()
		x.Platforms[e] = &plat
	case *components.Spawner:
		sp := c.Copy()
		x.Spawners[e] = &sp
	case *components.Sprite:
		spr := c.Copy()
		x.Sprites[e] = &spr
	case *components.Transform:
		tra := c.Copy()
		x.Transforms[e] = &tra
	case *components.Velocity:
		vel := c.Copy()
		x.Velocities[e] = &vel
	case *components.TimedLife:
		tl := c.Copy()
		x.TimedLives[e] = &tl
	case *components.Hitpoints:
		hp := c.Copy()
		x.Hitpoints[e] = &hp
	case *components.ContactDamage:
		cd := c.Copy()
		x.ContactDamages[e] = &cd
	}
}
