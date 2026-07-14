package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"encoding/gob"
	"fmt"
	"log"
	"maps"
	"math/rand/v2"
	"slices"
)

// Do not instantiate directly, use NewECS()
type ECSContainer struct {
	nextEntity           common.EntityId
	scheduledForDeletion []common.EntityId

	InputLog     map[uint64]map[common.EntityId]InputState
	Camera       utils.Vec2f
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
	Deathrattles      Storage[deathrattle]
	FloatingTexts     Storage[floatingText]
	ChunkLoaders      Storage[chunkLoader]

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
	DeathrattleManager      deathrattleManager
	FloatingTextManager     floatingTextManager
	ChunkLoaderManager      chunkLoaderManager
}

func NewECSContainer() *ECSContainer {
	gob.Register(&inputDto{})
	gob.Register(&parentDto{})
	gob.Register(&transformDto{})
	gob.Register(&velocityDto{})
	gob.Register(&spriteDto{})
	gob.Register(&animationDto{})
	gob.Register(&physicsColliderDto{})
	gob.Register(&platformColliderDto{})
	gob.Register(&hitboxColliderDto{})
	gob.Register(&hurtboxColliderDto{})
	gob.Register(&spawnerDto{})
	gob.Register(&timerDto{})
	gob.Register(&hitpointsDto{})
	gob.Register(&contactDamageDto{})
	gob.Register(&abilitiesDto{})
	gob.Register(&facePositionDto{})
	gob.Register(&equipmentDto{})
	gob.Register(&equipperDto{})
	gob.Register(&deathrattleDto{})
	gob.Register(&floatingTextDto{})
	gob.Register(&chunkLoaderDto{})
	gob.Register(&shapes.CircleParams{})
	gob.Register(&shapes.RectangleParams{})
	gob.Register(&shapes.PolygonParams{})
	gob.Register(&DodgeParams{})
	gob.Register(&ExplodeParams{})
	gob.Register(&InputFollowParams{})
	gob.Register(&InputDemoParams{})
	gob.Register(&InputLoopParams{})
	gob.Register(&InputReplayParams{})

	return &ECSContainer{
		nextEntity: 0,
		InputLog:   make(map[uint64]map[common.EntityId]InputState),

		Rng: rand.New(rand.NewPCG(data.RngSeed1, data.RngSeed2)),

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
		Deathrattles:      Storage[deathrattle]{order: []common.EntityId{}, data: make(map[common.EntityId]*deathrattle)},
		FloatingTexts:     Storage[floatingText]{order: []common.EntityId{}, data: make(map[common.EntityId]*floatingText)},
		ChunkLoaders:      Storage[chunkLoader]{order: []common.EntityId{}, data: make(map[common.EntityId]*chunkLoader)},

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
		DeathrattleManager:      deathrattleManager{},
		FloatingTextManager:     floatingTextManager{},
		ChunkLoaderManager:      chunkLoaderManager{},
	}
}

func (x *ECSContainer) GetCurrentTickInputs() (map[common.EntityId]InputState, error) {
	if tickInputs, ok := x.InputLog[x.TickIdx]; ok {
		return tickInputs, nil
	}
	return nil, fmt.Errorf("no inputs found for tick %d", x.TickIdx)
}

func (x *ECSContainer) GetCurrentTickInputsForEntity(e common.EntityId) (InputState, error) {
	if tickInputs, ok := x.InputLog[x.TickIdx]; ok {
		if input, ok := tickInputs[e]; ok {
			return input, nil
		}
		return InputState{}, fmt.Errorf("no input found for entity %d in tick %d", e, x.TickIdx)
	}
	return InputState{}, fmt.Errorf("no inputs found for tick %d", x.TickIdx)
}

func (x *ECSContainer) SetTickInputs(tick uint64, inputs map[common.EntityId]InputState) {
	x.InputLog[tick] = inputs
}

func (x *ECSContainer) AddEmptyEntity() common.EntityId {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *ECSContainer) AddEntity(comps ...Component) common.EntityId {
	e := x.AddEmptyEntity()

	for _, comp := range comps {
		x.AddComponent(e, comp)
	}

	return e
}

func (x *ECSContainer) ScheduleRemoveEntity(e common.EntityId) {
	if !slices.Contains(x.scheduledForDeletion, e) {
		x.scheduledForDeletion = append(x.scheduledForDeletion, e)
	}
}

func (x *ECSContainer) RemoveScheduledEntities() error {
	for _, e := range slices.Clone(x.scheduledForDeletion) {
		if x.Deathrattles.HasComponent(e) {
			err := x.DeathrattleManager.Effect(e, x)
			if err != nil {
				log.Printf("Error executing deathrattle for entity %d: %v\n", e, err)
			}
		}

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
		x.Deathrattles.deleteEntity(e)
		x.FloatingTexts.deleteEntity(e)
		x.ChunkLoaders.deleteEntity(e)

		pm := parentManager{}
		err := pm.RemoveParentFromAllEntities(e, x)
		if err != nil {
			log.Printf("error removing entity %d from parent component of all entities: %v", e, err)
			continue
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

		maps.DeleteFunc(x.TickState.CollisionGrid, func(k utils.Vec2i, v []common.EntityId) bool {
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

		x.scheduledForDeletion = slices.DeleteFunc(x.scheduledForDeletion, func(v common.EntityId) bool {
			return v == e
		})
	}
	return nil
}

func (x *ECSContainer) AddComponent(e common.EntityId, comp Component) {
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
	case *deathrattle:
		x.Deathrattles.addComponent(e, c.Copy())
	case *floatingText:
		x.FloatingTexts.addComponent(e, c.Copy())
	case *chunkLoader:
		x.ChunkLoaders.addComponent(e, c.Copy())
	default:
		log.Printf("warning: attempted to add component of type %T to entity %d, but no case for that component type exists in ECS.AddComponent\n", comp, e)
	}
}

func (x *ECSContainer) GetEntityComponents(eId common.EntityId) []ComponentDto {
	var comps []ComponentDto

	if c, err := x.Parents.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Transforms.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Velocities.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Sprites.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Animations.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.PhysicsColliders.getComponent(eId); err == nil {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting physics collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, dto)
	}
	if c, err := x.PlatformColliders.getComponent(eId); err == nil {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting platform collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, dto)
	}
	if c, err := x.HitboxColliders.getComponent(eId); err == nil {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting hitbox collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, dto)
	}
	if c, err := x.HurtboxColliders.getComponent(eId); err == nil {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting hurtbox collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, dto)
	}
	if c, err := x.Spawners.getComponent(eId); err == nil {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting spawner component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, dto)
	}
	if c, err := x.Timers.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Hitpoints.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.ContactDamages.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Inputs.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Abilities.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.FacePositions.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Equipments.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Equippers.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.Deathrattles.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.FloatingTexts.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}
	if c, err := x.ChunkLoaders.getComponent(eId); err == nil {
		dto := c.ToDto()
		comps = append(comps, dto)
	}

	return comps
}

func DtoToComponent[T ComponentDto](dto T) (Component, error) {
	switch d := any(dto).(type) {
	case *parentDto:
		return d.ToComponent(), nil
	case *transformDto:
		return d.ToComponent(), nil
	case *velocityDto:
		return d.ToComponent(), nil
	case *spriteDto:
		return d.ToComponent(), nil
	case *animationDto:
		return d.ToComponent(), nil
	case *physicsColliderDto:
		dto, err := d.ToComponent()
		if err != nil {
			return nil, fmt.Errorf("error converting physics collider DTO to component: %v", err)
		}
		return dto, nil
	case *platformColliderDto:
		dto, err := d.ToComponent()
		if err != nil {
			return nil, fmt.Errorf("error converting platform collider DTO to component: %v", err)
		}
		return dto, nil
	case *hitboxColliderDto:
		dto, err := d.ToComponent()
		if err != nil {
			return nil, fmt.Errorf("error converting hitbox collider DTO to component: %v", err)
		}
		return dto, nil
	case *hurtboxColliderDto:
		dto, err := d.ToComponent()
		if err != nil {
			return nil, fmt.Errorf("error converting hurtbox collider DTO to component: %v", err)
		}
		return dto, nil
	case *spawnerDto:
		dto, err := d.ToComponent()
		if err != nil {
			return nil, fmt.Errorf("error converting spawner DTO to component: %v", err)
		}
		return dto, nil
	case *timerDto:
		return d.ToComponent(), nil
	case *hitpointsDto:
		return d.ToComponent(), nil
	case *contactDamageDto:
		return d.ToComponent(), nil
	case *inputDto:
		return d.ToComponent(), nil
	case *abilitiesDto:
		return d.ToComponent(), nil
	case *facePositionDto:
		return d.ToComponent(), nil
	case *equipmentDto:
		return d.ToComponent(), nil
	case *equipperDto:
		return d.ToComponent(), nil
	case *deathrattleDto:
		return d.ToComponent(), nil
	case *floatingTextDto:
		return d.ToComponent(), nil
	case *chunkLoaderDto:
		return d.ToComponent(), nil
	default:
		return nil, fmt.Errorf("unknown component DTO type: %T", dto)
	}
}
