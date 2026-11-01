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
	scheduledForDeath    []common.EntityId

	tickInputs     map[common.EntityId]InputState
	tickInputsTick uint64

	Camera       utils.Vec2f
	CameraFollow bool

	Rng       *rand.Rand
	rngSource *rand.PCG
	TickIdx   uint64
	TickState common.TickState

	isSimulatedPos func(worldPos utils.Vec2f) bool

	storages []componentStorage

	abilityFuncs     map[AbilityEnum]AbilityFunc
	timerFuncs       map[TimerFuncEnum]TimerFunc
	inputSourceFuncs map[InputTypeEnum]InputSourceFunc

	Inputs            Storage[input]
	Hierarchies       Storage[hierarchy]
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
	Persistents       Storage[persistent]
	Tethers           Storage[tether]

	InputManager            inputManager
	HierarchyManager        hierarchyManager
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
	PersistentManager       persistentManager
	TetherManager           tetherManager
}

func NewECSContainer() *ECSContainer {
	gob.Register(&inputDto{})
	gob.Register(&hierarchyDto{})
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
	gob.Register(&persistentDto{})
	gob.Register(&tetherDto{})

	gob.Register(&shapes.CircleParams{})
	gob.Register(&shapes.RectangleParams{})
	gob.Register(&shapes.PolygonParams{})

	rngSource := rand.NewPCG(data.RngSeed1, data.RngSeed2)

	x := &ECSContainer{
		nextEntity: 0,

		abilityFuncs:     make(map[AbilityEnum]AbilityFunc),
		timerFuncs:       make(map[TimerFuncEnum]TimerFunc),
		inputSourceFuncs: make(map[InputTypeEnum]InputSourceFunc),

		Rng:       rand.New(rngSource),
		rngSource: rngSource,

		TickIdx:   0,
		TickState: common.TickState{},

		Inputs:            newStorage[input](),
		Hierarchies:       newStorage[hierarchy](),
		Transforms:        newStorage[transform](),
		Velocities:        newStorage[velocity](),
		Sprites:           newStorage[sprite](),
		Animations:        newStorage[animation](),
		PhysicsColliders:  newStorage[physicsCollider](),
		PlatformColliders: newStorage[platformCollider](),
		HitboxColliders:   newStorage[hitboxCollider](),
		HurtboxColliders:  newStorage[hurtboxCollider](),
		Spawners:          newStorage[spawner](),
		Timers:            newStorage[timer](),
		Hitpoints:         newStorage[hitpoints](),
		ContactDamages:    newStorage[contactDamage](),
		Abilities:         newStorage[abilities](),
		FacePositions:     newStorage[facePosition](),
		Equipments:        newStorage[equipment](),
		Equippers:         newStorage[equipper](),
		Deathrattles:      newStorage[deathrattle](),
		FloatingTexts:     newStorage[floatingText](),
		ChunkLoaders:      newStorage[chunkLoader](),
		Persistents:       newStorage[persistent](),
		Tethers:           newStorage[tether](),

		InputManager:            inputManager{},
		HierarchyManager:        hierarchyManager{},
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
		PersistentManager:       persistentManager{},
		TetherManager:           tetherManager{},
	}

	// The order decides the order of an entity's components in saved data
	x.storages = []componentStorage{
		&x.Hierarchies,
		&x.Transforms,
		&x.Velocities,
		&x.Sprites,
		&x.Animations,
		&x.PhysicsColliders,
		&x.PlatformColliders,
		&x.HitboxColliders,
		&x.HurtboxColliders,
		&x.Spawners,
		&x.Timers,
		&x.Hitpoints,
		&x.ContactDamages,
		&x.Inputs,
		&x.Abilities,
		&x.FacePositions,
		&x.Equipments,
		&x.Equippers,
		&x.Deathrattles,
		&x.FloatingTexts,
		&x.ChunkLoaders,
		&x.Persistents,
		&x.Tethers,
	}

	return x
}

func (x *ECSContainer) GetCurrentTickInputs() (map[common.EntityId]InputState, error) {
	if x.tickInputs == nil || x.tickInputsTick != x.TickIdx {
		return nil, fmt.Errorf("no inputs found for tick %d", x.TickIdx)
	}
	return x.tickInputs, nil
}

func (x *ECSContainer) GetCurrentTickInputsForEntity(e common.EntityId) (InputState, error) {
	tickInputs, err := x.GetCurrentTickInputs()
	if err != nil {
		return InputState{}, err
	}

	input, ok := tickInputs[e]
	if !ok {
		return InputState{}, fmt.Errorf("no input found for entity %d in tick %d", e, x.TickIdx)
	}
	return input, nil
}

func (x *ECSContainer) SetTickInputs(tick uint64, inputs map[common.EntityId]InputState) {
	x.tickInputs = inputs
	x.tickInputsTick = tick
}

func (x *ECSContainer) RngState() ([]byte, error) {
	return x.rngSource.MarshalBinary()
}

func (x *ECSContainer) SetRngState(state []byte) error {
	return x.rngSource.UnmarshalBinary(state)
}

func (x *ECSContainer) NextEntity() common.EntityId {
	return x.nextEntity
}

func (x *ECSContainer) SetNextEntity(e common.EntityId) {
	x.nextEntity = e
}

func (x *ECSContainer) AddEmptyEntity() common.EntityId {
	x.nextEntity++
	return x.nextEntity - 1
}

func (x *ECSContainer) SetSimulatedPosCheck(isSimulatedPos func(worldPos utils.Vec2f) bool) {
	x.isSimulatedPos = isSimulatedPos
}

func (x *ECSContainer) AddNewEntity(comps ...Component) (common.EntityId, error) {
	if err := x.checkSpawnPos(comps); err != nil {
		return -1, err
	}

	e := x.AddEmptyEntity()

	for _, comp := range comps {
		x.AddComponent(e, comp)
	}

	return e, nil
}

// An entity spawned outside the simulated area would never be simulated, only stored or despawned at the end of
// the tick. Chunkloaders are exempt, since they decide what is simulated.
func (x *ECSContainer) checkSpawnPos(comps []Component) error {
	if x.isSimulatedPos == nil {
		return nil
	}

	var t *transform
	for _, comp := range comps {
		switch c := comp.(type) {
		case *chunkLoader:
			return nil
		case *transform:
			t = c
		}
	}

	// A new entity has no parent yet, so its local position is its world position
	if t != nil && !x.isSimulatedPos(t.pos) {
		return fmt.Errorf("cannot spawn entity at %v, outside the active chunks", t.pos)
	}

	return nil
}

func (x *ECSContainer) AddEntityWithId(eId common.EntityId, comps ...Component) error {
	if x.HasEntity(eId) {
		return fmt.Errorf("entity %d already exists", eId)
	}

	if eId >= x.nextEntity {
		log.Printf("Warning: Adding entity with ID %d, which isn't below nextEntity %d. Updating nextEntity to %d\n", eId, x.nextEntity, eId+1)
		x.nextEntity = eId + 1
	}

	for _, comp := range comps {
		x.AddComponent(eId, comp)
	}

	return nil
}

func (x *ECSContainer) ScheduleRemoveEntity(e common.EntityId) {
	if !slices.Contains(x.scheduledForDeletion, e) {
		x.scheduledForDeletion = append(x.scheduledForDeletion, e)
	}
}

func (x *ECSContainer) ScheduleKillEntity(e common.EntityId) {
	x.ScheduleRemoveEntity(e)
	if !slices.Contains(x.scheduledForDeath, e) {
		x.scheduledForDeath = append(x.scheduledForDeath, e)
	}
}

func (x *ECSContainer) RemoveScheduledEntities() error {
	// Indexed loop, since death effects may schedule further deaths
	for i := 0; i < len(x.scheduledForDeath); i++ {
		e := x.scheduledForDeath[i]
		if x.Deathrattles.HasComponent(e) {
			err := x.DeathrattleManager.Effect(e, x)
			if err != nil {
				log.Printf("Error executing deathrattle for entity %d: %v\n", e, err)
			}
		}
	}

	killed := make(map[common.EntityId]struct{}, len(x.scheduledForDeath))
	for _, e := range x.scheduledForDeath {
		killed[e] = struct{}{}
	}
	x.scheduledForDeath = x.scheduledForDeath[:0]

	for _, e := range slices.Clone(x.scheduledForDeletion) {
		_, wasKilled := killed[e]
		x.removeEntity(e, wasKilled)
	}

	return nil
}

func (x *ECSContainer) removeEntity(e common.EntityId, dropHeld bool) {
	hm := hierarchyManager{}

	// Children are resolved before this entity is deleted, since dropping what they hold needs their ancestors' world
	// transforms
	for _, c := range hm.GetChildren(e, x) {
		linkKind := hm.GetLinkKind(c, x)

		if linkKind == Link_Attached || (linkKind == Link_Held && dropHeld) {
			if err := hm.Detach(c, x); err != nil {
				log.Printf("error detaching entity %d from removed entity %d: %v", c, e, err)
			}
			continue
		}

		// Parts of a killed entity are removed rather than killed, but still drop what they hold
		x.removeEntity(c, dropHeld && linkKind == Link_Part)
	}

	removeChild(hm.GetParent(e, x), e, x)

	for _, s := range x.storages {
		s.deleteEntity(e)
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

func (x *ECSContainer) AddComponent(e common.EntityId, comp Component) {
	comp.addTo(x, e)
}

func (x *ECSContainer) HasEntity(e common.EntityId) bool {
	for _, s := range x.storages {
		if s.HasComponent(e) {
			return true
		}
	}

	return false
}

func (x *ECSContainer) GetEntityComponents(eId common.EntityId) []ComponentDto {
	var comps []ComponentDto

	for _, s := range x.storages {
		dto, ok, err := s.toDto(eId)
		if err != nil {
			log.Printf("error converting component of entity %d to DTO: %v", eId, err)
			continue
		}
		if ok {
			comps = append(comps, dto)
		}
	}

	return comps
}

// Returns the DTOs of root, its held entities and its persistent parts, which is the unit that gets stored.
// Links to entities outside the unit are cut, and root's transform becomes a world transform if it was attached.
func (x *ECSContainer) GetCompositionComponents(root common.EntityId) (map[common.EntityId][]ComponentDto, error) {
	hm := hierarchyManager{}
	tm := transformManager{}

	members := make(map[common.EntityId][]ComponentDto)
	x.collectComposition(root, members)

	rootIsAttached := hm.GetParent(root, x) != -1

	for e, dtos := range members {
		for _, dto := range dtos {
			switch d := dto.(type) {
			case *hierarchyDto:
				d.Children = slices.DeleteFunc(d.Children, func(c common.EntityId) bool {
					_, ok := members[c]
					return !ok
				})
				if e == root {
					d.Parent = -1
				}
			case *equipperDto:
				for slot, eq := range d.Equipment {
					if _, ok := members[eq]; !ok {
						log.Printf("consistency: equipper %d is stored without its equipment %d in slot %v", e, eq, slot)
					}
				}
			case *transformDto:
				if e != root || !rootIsAttached {
					continue
				}

				var err error
				if d.Pos, err = tm.GetWorldPos(e, x); err != nil {
					return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
				}
				if d.Rotation, err = tm.GetWorldRotation(e, x); err != nil {
					return nil, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
				}
				if d.Scale, err = tm.GetWorldScale(e, x); err != nil {
					return nil, fmt.Errorf("error getting world scale of entity %d: %v", e, err)
				}
			}
		}
	}

	return members, nil
}

func (x *ECSContainer) collectComposition(e common.EntityId, members map[common.EntityId][]ComponentDto) {
	hm := hierarchyManager{}

	members[e] = x.GetEntityComponents(e)

	for _, c := range hm.GetChildren(e, x) {
		switch hm.GetLinkKind(c, x) {
		case Link_Attached:
			continue
		case Link_Part:
			if !x.Persistents.HasComponent(c) {
				continue
			}
		}
		x.collectComposition(c, members)
	}
}

func DtoToComponent(dto ComponentDto) (Component, error) {
	return dto.toComponent()
}

func ComponentToDto(comp Component) (ComponentDto, error) {
	return comp.toDto()
}
