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

	InputLog     map[uint64]map[common.EntityId]InputState
	Camera       utils.Vec2f
	CameraFollow bool

	Rng       *rand.Rand
	rngSource *rand.PCG
	TickIdx   uint64
	TickState common.TickState

	isSimulatedPos func(worldPos utils.Vec2f) bool

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

	return &ECSContainer{
		nextEntity: 0,
		InputLog:   make(map[uint64]map[common.EntityId]InputState),

		abilityFuncs:     make(map[AbilityEnum]AbilityFunc),
		timerFuncs:       make(map[TimerFuncEnum]TimerFunc),
		inputSourceFuncs: make(map[InputTypeEnum]InputSourceFunc),

		Rng:       rand.New(rngSource),
		rngSource: rngSource,

		TickIdx:   0,
		TickState: common.TickState{},

		Inputs:            Storage[input]{order: []common.EntityId{}, data: make(map[common.EntityId]*input)},
		Hierarchies:       Storage[hierarchy]{order: []common.EntityId{}, data: make(map[common.EntityId]*hierarchy)},
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
		Persistents:       Storage[persistent]{order: []common.EntityId{}, data: make(map[common.EntityId]*persistent)},
		Tethers:           Storage[tether]{order: []common.EntityId{}, data: make(map[common.EntityId]*tether)},

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

	x.Hierarchies.deleteEntity(e)
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
	x.Persistents.deleteEntity(e)
	x.Tethers.deleteEntity(e)

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
	switch c := comp.(type) {
	case *hierarchy:
		x.Hierarchies.addComponent(e, c.Copy())
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
	case *persistent:
		x.Persistents.addComponent(e, c.Copy())
	case *tether:
		x.Tethers.addComponent(e, c.Copy())
	default:
		log.Printf("warning: attempted to add component of type %T to entity %d, but no case for that component type exists in ECS.AddComponent\n", comp, e)
	}
}

func (x *ECSContainer) HasEntity(e common.EntityId) bool {
	return x.Inputs.HasComponent(e) ||
		x.Hierarchies.HasComponent(e) ||
		x.Transforms.HasComponent(e) ||
		x.Velocities.HasComponent(e) ||
		x.Sprites.HasComponent(e) ||
		x.Animations.HasComponent(e) ||
		x.PhysicsColliders.HasComponent(e) ||
		x.PlatformColliders.HasComponent(e) ||
		x.HitboxColliders.HasComponent(e) ||
		x.HurtboxColliders.HasComponent(e) ||
		x.Spawners.HasComponent(e) ||
		x.Timers.HasComponent(e) ||
		x.Hitpoints.HasComponent(e) ||
		x.ContactDamages.HasComponent(e) ||
		x.Abilities.HasComponent(e) ||
		x.FacePositions.HasComponent(e) ||
		x.Equipments.HasComponent(e) ||
		x.Equippers.HasComponent(e) ||
		x.Deathrattles.HasComponent(e) ||
		x.FloatingTexts.HasComponent(e) ||
		x.ChunkLoaders.HasComponent(e) ||
		x.Persistents.HasComponent(e) ||
		x.Tethers.HasComponent(e)
}

func (x *ECSContainer) GetEntityComponents(eId common.EntityId) []ComponentDto {
	var comps []ComponentDto

	if c, ok := x.Hierarchies.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Transforms.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Velocities.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Sprites.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Animations.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.PhysicsColliders.tryGetComponent(eId); ok {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting physics collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, &dto)
	}
	if c, ok := x.PlatformColliders.tryGetComponent(eId); ok {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting platform collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, &dto)
	}
	if c, ok := x.HitboxColliders.tryGetComponent(eId); ok {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting hitbox collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, &dto)
	}
	if c, ok := x.HurtboxColliders.tryGetComponent(eId); ok {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting hurtbox collider component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, &dto)
	}
	if c, ok := x.Spawners.tryGetComponent(eId); ok {
		dto, err := c.ToDto()
		if err != nil {
			log.Printf("error converting spawner component of entity %d to DTO: %v", eId, err)
		}
		comps = append(comps, &dto)
	}
	if c, ok := x.Timers.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Hitpoints.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.ContactDamages.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Inputs.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Abilities.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.FacePositions.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Equipments.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Equippers.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Deathrattles.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.FloatingTexts.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.ChunkLoaders.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Persistents.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
	}
	if c, ok := x.Tethers.tryGetComponent(eId); ok {
		dto := c.ToDto()
		comps = append(comps, &dto)
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

func DtoToComponent[T ComponentDto](dto T) (Component, error) {
	switch d := any(dto).(type) {
	case *hierarchyDto:
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
	case *persistentDto:
		return d.ToComponent(), nil
	case *tetherDto:
		return d.ToComponent(), nil
	default:
		return nil, fmt.Errorf("unknown component DTO type: %T", dto)
	}
}

func ComponentToDto(comp Component) (ComponentDto, error) {
	switch c := comp.(type) {
	case *hierarchy:
		dto := c.ToDto()
		return &dto, nil
	case *transform:
		dto := c.ToDto()
		return &dto, nil
	case *velocity:
		dto := c.ToDto()
		return &dto, nil
	case *sprite:
		dto := c.ToDto()
		return &dto, nil
	case *animation:
		dto := c.ToDto()
		return &dto, nil
	case *physicsCollider:
		dto, err := c.ToDto()
		if err != nil {
			return nil, fmt.Errorf("error converting physics collider component to DTO: %v", err)
		}
		return &dto, nil
	case *platformCollider:
		dto, err := c.ToDto()
		if err != nil {
			return nil, fmt.Errorf("error converting platform collider component to DTO: %v", err)
		}
		return &dto, nil
	case *hitboxCollider:
		dto, err := c.ToDto()
		if err != nil {
			return nil, fmt.Errorf("error converting hitbox collider component to DTO: %v", err)
		}
		return &dto, nil
	case *hurtboxCollider:
		dto, err := c.ToDto()
		if err != nil {
			return nil, fmt.Errorf("error converting hurtbox collider component to DTO: %v", err)
		}
		return &dto, nil
	case *spawner:
		dto, err := c.ToDto()
		if err != nil {
			return nil, fmt.Errorf("error converting spawner component to DTO: %v", err)
		}
		return &dto, nil
	case *timer:
		dto := c.ToDto()
		return &dto, nil
	case *hitpoints:
		dto := c.ToDto()
		return &dto, nil
	case *contactDamage:
		dto := c.ToDto()
		return &dto, nil
	case *input:
		dto := c.ToDto()
		return &dto, nil
	case *abilities:
		dto := c.ToDto()
		return &dto, nil
	case *facePosition:
		dto := c.ToDto()
		return &dto, nil
	case *equipment:
		dto := c.ToDto()
		return &dto, nil
	case *equipper:
		dto := c.ToDto()
		return &dto, nil
	case *deathrattle:
		dto := c.ToDto()
		return &dto, nil
	case *floatingText:
		dto := c.ToDto()
		return &dto, nil
	case *chunkLoader:
		dto := c.ToDto()
		return &dto, nil
	case *persistent:
		dto := c.ToDto()
		return &dto, nil
	case *tether:
		dto := c.ToDto()
		return &dto, nil
	default:
		return nil, fmt.Errorf("unknown component type: %T", comp)
	}
}
