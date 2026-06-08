package ecs

import (
	"ebittest/ecs/common"
	"fmt"
	"maps"
)

type World struct {
	nextEntity common.EntityId

	TickIdx   uint64
	TickState common.TickState

	Inputs            map[common.EntityId]*input
	Parents           map[common.EntityId]*parent
	Transforms        map[common.EntityId]*transform
	Velocities        map[common.EntityId]*velocity
	Sprites           map[common.EntityId]*sprite
	Animations        map[common.EntityId]*animation
	CollisionLayers   map[common.EntityId]*collisionLayer
	PhysicsColliders  map[common.EntityId]*physicsCollider
	PlatformColliders map[common.EntityId]*platformCollider
	HitboxColliders   map[common.EntityId]*hitboxCollider
	HurtboxColliders  map[common.EntityId]*hurtboxCollider
	Spawners          map[common.EntityId]*spawner
	TimedLives        map[common.EntityId]*timedLife
	Hitpoints         map[common.EntityId]*hitpoints
	ContactDamages    map[common.EntityId]*contactDamage
}

func NewWorld() *World {
	return &World{
		nextEntity: 0,

		TickIdx:   0,
		TickState: common.TickState{},

		Inputs:            make(map[common.EntityId]*input),
		Parents:           make(map[common.EntityId]*parent),
		Transforms:        make(map[common.EntityId]*transform),
		Velocities:        make(map[common.EntityId]*velocity),
		Sprites:           make(map[common.EntityId]*sprite),
		Animations:        make(map[common.EntityId]*animation),
		CollisionLayers:   make(map[common.EntityId]*collisionLayer),
		PhysicsColliders:  make(map[common.EntityId]*physicsCollider),
		PlatformColliders: make(map[common.EntityId]*platformCollider),
		HitboxColliders:   make(map[common.EntityId]*hitboxCollider),
		HurtboxColliders:  make(map[common.EntityId]*hurtboxCollider),
		Spawners:          make(map[common.EntityId]*spawner),
		TimedLives:        make(map[common.EntityId]*timedLife),
		Hitpoints:         make(map[common.EntityId]*hitpoints),
		ContactDamages:    make(map[common.EntityId]*contactDamage),
	}
}

func (x *World) AddEmptyEntity() common.EntityId {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *World) AddEntity(comps ...component) common.EntityId {
	e := x.AddEmptyEntity()

	for _, comp := range comps {
		x.AddComponent(e, comp)
	}

	return e
}

func (x *World) RemoveEntity(e common.EntityId) error {
	delete(x.Parents, e)
	delete(x.Transforms, e)
	delete(x.Velocities, e)
	delete(x.Sprites, e)
	delete(x.Animations, e)
	delete(x.CollisionLayers, e)
	delete(x.PhysicsColliders, e)
	delete(x.PlatformColliders, e)
	delete(x.HitboxColliders, e)
	delete(x.HurtboxColliders, e)
	delete(x.Spawners, e)
	delete(x.TimedLives, e)
	delete(x.Hitpoints, e)
	delete(x.ContactDamages, e)

	pm := ParentManager{}
	err := pm.RemoveParentFromAllEntities(e, x.Parents, x.Transforms)
	if err != nil {
		return fmt.Errorf("error removing entity %d from parent component of all entities: %v", e, err)
	}

	maps.DeleteFunc(x.TickState.AABBCollisions, func(k common.EntityId, v []common.EntityId) bool {
		if k == e {
			return true
		}
		for _, vE := range v {
			if vE == e {
				return true
			}
		}
		return false
	})

	maps.DeleteFunc(x.TickState.CollisionGrid, func(k common.CellKey, v []common.EntityId) bool {
		for _, vE := range v {
			if vE == e {
				return true
			}
		}
		return false
	})

	maps.DeleteFunc(x.TickState.Collisions, func(k common.EntityId, v map[common.EntityId]common.Collision) bool {
		for vE, _ := range v {
			if vE == e {
				return true
			}
		}
		return false
	})

	maps.DeleteFunc(x.TickState.ProximateEntities, func(k common.EntityId, v []common.EntityId) bool {
		for _, vE := range v {
			if vE == e {
				return true
			}
		}
		return false
	})

	return nil
}

func (x *World) AddComponent(e common.EntityId, comp component) {
	switch c := comp.(type) {
	case *physicsCollider:
		col := c.Copy()
		x.PhysicsColliders[e] = &col
	case *collisionLayer:
		cl := c.Copy()
		x.CollisionLayers[e] = &cl
	case *hitboxCollider:
		hb := c.Copy()
		x.HitboxColliders[e] = &hb
	case *hurtboxCollider:
		hb := c.Copy()
		x.HurtboxColliders[e] = &hb
	case *platformCollider:
		pc := c.Copy()
		x.PlatformColliders[e] = &pc
	case *input:
		inp := c.Copy()
		x.Inputs[e] = &inp
	case *parent:
		par := c.Copy()
		x.Parents[e] = &par
	case *spawner:
		sp := c.Copy()
		x.Spawners[e] = &sp
	case *sprite:
		spr := c.Copy()
		x.Sprites[e] = &spr
	case *animation:
		anim := c.Copy()
		x.Animations[e] = &anim
	case *transform:
		tra := c.Copy()
		x.Transforms[e] = &tra
	case *velocity:
		vel := c.Copy()
		x.Velocities[e] = &vel
	case *timedLife:
		tl := c.Copy()
		x.TimedLives[e] = &tl
	case *hitpoints:
		hp := c.Copy()
		x.Hitpoints[e] = &hp
	case *contactDamage:
		cd := c.Copy()
		x.ContactDamages[e] = &cd
	}
}
