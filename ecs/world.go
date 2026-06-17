package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"maps"
	"math/rand/v2"
	"slices"
)

type Storage[T component] struct {
	order []common.EntityId
	data  map[common.EntityId]*T
}

func (x *Storage[T]) GetEntities() []common.EntityId {
	return x.order
}

func (x *Storage[T]) HasComponent(e common.EntityId) bool {
	_, ok := x.data[e]
	return ok
}

func (x *Storage[T]) getData() map[common.EntityId]*T {
	return x.data
}

func (x *Storage[T]) getComponent(e common.EntityId) (*T, error) {
	c, ok := x.data[e]
	if !ok {
		return nil, fmt.Errorf("could not get %T component of entity %d", *new(T), e)
	}

	return c, nil
}

func (x *Storage[T]) deleteEntity(e common.EntityId) {
	x.order = slices.DeleteFunc(x.order, func(id common.EntityId) bool {
		return id == e
	})
	delete(x.getData(), e)
}

func (x *Storage[T]) addComponent(e common.EntityId, c T) {
	x.order = append(x.order, e)
	x.data[e] = &c
}

// TODO: Move manager instances here instead of instantiating every time
// Do not instantiate directly, use NewWorld()
type World struct {
	nextEntity   common.EntityId
	InputLog     map[uint64]map[common.EntityId]InputState
	Camera       utils.Vec2
	CameraFollow bool

	Rng       *rand.Rand
	TickIdx   uint64
	TickState common.TickState

	Inputs            Storage[input]
	Parents           Storage[parent]
	Transforms        Storage[transform]
	Velocities        Storage[velocity]
	Sprites           Storage[sprite]
	Animations        Storage[animation]
	PhysicsColliders  Storage[physicsCollider]
	PlatformColliders Storage[platformCollider]
	HitboxColliders   Storage[hitboxCollider]
	HurtboxColliders  Storage[hurtboxCollider]
	Spawners          Storage[spawner]
	Timers            Storage[timer]
	Hitpoints         Storage[hitpoints]
	ContactDamages    Storage[contactDamage]
	Abilities         Storage[abilities]
	FacePositions     Storage[facePosition]
	Equipments        Storage[equipment]
	Equippers         Storage[equipper]

	InputManager            inputManager
	ParentManager           parentManager
	TransformManager        transformManager
	VelocityManager         velocityManager
	SpriteManager           spriteManager
	AnimationManager        animationManager
	PhysicsColliderManager  physicsColliderManager
	PlatformColliderManager platformColliderManager
	HitboxColliderManager   hitboxColliderManager
	HurtboxColliderManager  hurtboxColliderManager
	SpawnerManager          spawnerManager
	TimerManager            timerManager
	HitpointsManager        hitpointsManager
	ContactDamageManager    contactDamageManager
	AbilitiesManager        abilitiesManager
	FacePositionManager     facePositionManager
	EquipManager            equipManager
}

func NewWorld() *World {
	return &World{
		nextEntity: 0,
		InputLog:   make(map[uint64]map[common.EntityId]InputState),

		Rng: rand.New(rand.NewChaCha8([32]byte{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
		})),

		TickIdx:   0,
		TickState: common.TickState{},

		Inputs:            Storage[input]{order: []common.EntityId{}, data: make(map[common.EntityId]*input)},
		Parents:           Storage[parent]{order: []common.EntityId{}, data: make(map[common.EntityId]*parent)},
		Transforms:        Storage[transform]{order: []common.EntityId{}, data: make(map[common.EntityId]*transform)},
		Velocities:        Storage[velocity]{order: []common.EntityId{}, data: make(map[common.EntityId]*velocity)},
		Sprites:           Storage[sprite]{order: []common.EntityId{}, data: make(map[common.EntityId]*sprite)},
		Animations:        Storage[animation]{order: []common.EntityId{}, data: make(map[common.EntityId]*animation)},
		PhysicsColliders:  Storage[physicsCollider]{order: []common.EntityId{}, data: make(map[common.EntityId]*physicsCollider)},
		PlatformColliders: Storage[platformCollider]{order: []common.EntityId{}, data: make(map[common.EntityId]*platformCollider)},
		HitboxColliders:   Storage[hitboxCollider]{order: []common.EntityId{}, data: make(map[common.EntityId]*hitboxCollider)},
		HurtboxColliders:  Storage[hurtboxCollider]{order: []common.EntityId{}, data: make(map[common.EntityId]*hurtboxCollider)},
		Spawners:          Storage[spawner]{order: []common.EntityId{}, data: make(map[common.EntityId]*spawner)},
		Timers:            Storage[timer]{order: []common.EntityId{}, data: make(map[common.EntityId]*timer)},
		Hitpoints:         Storage[hitpoints]{order: []common.EntityId{}, data: make(map[common.EntityId]*hitpoints)},
		ContactDamages:    Storage[contactDamage]{order: []common.EntityId{}, data: make(map[common.EntityId]*contactDamage)},
		Abilities:         Storage[abilities]{order: []common.EntityId{}, data: make(map[common.EntityId]*abilities)},
		FacePositions:     Storage[facePosition]{order: []common.EntityId{}, data: make(map[common.EntityId]*facePosition)},
		Equipments:        Storage[equipment]{order: []common.EntityId{}, data: make(map[common.EntityId]*equipment)},
		Equippers:         Storage[equipper]{order: []common.EntityId{}, data: make(map[common.EntityId]*equipper)},

		InputManager:            inputManager{},
		ParentManager:           parentManager{},
		TransformManager:        transformManager{},
		VelocityManager:         velocityManager{},
		SpriteManager:           spriteManager{},
		AnimationManager:        animationManager{},
		PhysicsColliderManager:  physicsColliderManager{},
		PlatformColliderManager: platformColliderManager{},
		HitboxColliderManager:   hitboxColliderManager{},
		HurtboxColliderManager:  hurtboxColliderManager{},
		SpawnerManager:          spawnerManager{},
		TimerManager:            timerManager{},
		HitpointsManager:        hitpointsManager{},
		ContactDamageManager:    contactDamageManager{},
		AbilitiesManager:        abilitiesManager{},
		FacePositionManager:     facePositionManager{},
		EquipManager:            equipManager{},
	}
}

func (x *World) GetCurrentTickInputs() (map[common.EntityId]InputState, error) {
	if tickInputs, ok := x.InputLog[x.TickIdx]; ok {
		return tickInputs, nil
	}
	return nil, fmt.Errorf("no inputs found for tick %d", x.TickIdx)
}

func (x *World) GetCurrentTickInputsForEntity(e common.EntityId) (InputState, error) {
	if tickInputs, ok := x.InputLog[x.TickIdx]; ok {
		if input, ok := tickInputs[e]; ok {
			return input, nil
		}
		return InputState{}, fmt.Errorf("no input found for entity %d in tick %d", e, x.TickIdx)
	}
	return InputState{}, fmt.Errorf("no inputs found for tick %d", x.TickIdx)
}

func (x *World) SetTickInputs(tick uint64, inputs map[common.EntityId]InputState) {
	x.InputLog[tick] = inputs
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
	x.Parents.deleteEntity(e)
	x.Transforms.deleteEntity(e)
	x.Velocities.deleteEntity(e)
	x.Sprites.deleteEntity(e)
	x.Animations.deleteEntity(e)
	x.PhysicsColliders.deleteEntity(e)
	x.PlatformColliders.deleteEntity(e)
	x.HitboxColliders.deleteEntity(e)
	x.HurtboxColliders.deleteEntity(e)
	x.Spawners.deleteEntity(e)
	x.Timers.deleteEntity(e)
	x.Hitpoints.deleteEntity(e)
	x.ContactDamages.deleteEntity(e)
	x.Inputs.deleteEntity(e)
	x.Abilities.deleteEntity(e)
	x.FacePositions.deleteEntity(e)
	x.Equipments.deleteEntity(e)
	x.Equippers.deleteEntity(e)

	pm := parentManager{}
	err := pm.RemoveParentFromAllEntities(e, x)
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
		for vE := range v {
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
	case *parent:
		x.Parents.addComponent(e, c.Copy())
	case *transform:
		x.Transforms.addComponent(e, c.Copy())
	case *velocity:
		x.Velocities.addComponent(e, c.Copy())
	case *sprite:
		x.Sprites.addComponent(e, c.Copy())
	case *animation:
		x.Animations.addComponent(e, c.Copy())
	case *physicsCollider:
		x.PhysicsColliders.addComponent(e, c.Copy())
	case *platformCollider:
		x.PlatformColliders.addComponent(e, c.Copy())
	case *hitboxCollider:
		x.HitboxColliders.addComponent(e, c.Copy())
	case *hurtboxCollider:
		x.HurtboxColliders.addComponent(e, c.Copy())
	case *spawner:
		x.Spawners.addComponent(e, c.Copy())
	case *timer:
		x.Timers.addComponent(e, c.Copy())
	case *hitpoints:
		x.Hitpoints.addComponent(e, c.Copy())
	case *contactDamage:
		x.ContactDamages.addComponent(e, c.Copy())
	case *input:
		x.Inputs.addComponent(e, c.Copy())
	case *abilities:
		x.Abilities.addComponent(e, c.Copy())
	case *facePosition:
		x.FacePositions.addComponent(e, c.Copy())
	case *equipment:
		x.Equipments.addComponent(e, c.Copy())
	case *equipper:
		x.Equippers.addComponent(e, c.Copy())
	default:
		log.Printf("warning: attempted to add component of type %T to entity %d, but no case for that component type exists in World.AddComponent\n", comp, e)
	}
}
