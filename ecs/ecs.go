package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"encoding/gob"
	"fmt"
	"log"
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

	Inputs            Storage[Input]
	Hierarchies       Storage[Hierarchy]
	Transforms        Storage[Transform]
	Velocities        Storage[Velocity]
	Sprites           Storage[Sprite]
	Animations        Storage[Animation]
	PhysicsColliders  Storage[PhysicsCollider]
	PlatformColliders Storage[PlatformCollider]
	HitboxColliders   Storage[HitboxCollider]
	HurtboxColliders  Storage[HurtboxCollider]
	Spawners          Storage[Spawner]
	Timers            Storage[Timer]
	Hitpoints         Storage[Hitpoints]
	ContactDamages    Storage[ContactDamage]
	Abilities         Storage[Abilities]
	FacePositions     Storage[FacePosition]
	Equipments        Storage[Equipment]
	Equippers         Storage[Equipper]
	Deathrattles      Storage[Deathrattle]
	FloatingTexts     Storage[FloatingText]
	ChunkLoaders      Storage[ChunkLoader]
	Persistents       Storage[Persistent]
	Tethers           Storage[Tether]
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

	rngSource := rand.NewPCG(0, 0)

	x := &ECSContainer{
		nextEntity: 0,

		abilityFuncs:     make(map[AbilityEnum]AbilityFunc),
		timerFuncs:       make(map[TimerFuncEnum]TimerFunc),
		inputSourceFuncs: make(map[InputTypeEnum]InputSourceFunc),

		Rng:       rand.New(rngSource),
		rngSource: rngSource,

		TickIdx:   0,
		TickState: common.TickState{},

		Inputs:            newStorage[Input](),
		Hierarchies:       newStorage[Hierarchy](),
		Transforms:        newStorage[Transform](),
		Velocities:        newStorage[Velocity](),
		Sprites:           newStorage[Sprite](),
		Animations:        newStorage[Animation](),
		PhysicsColliders:  newStorage[PhysicsCollider](),
		PlatformColliders: newStorage[PlatformCollider](),
		HitboxColliders:   newStorage[HitboxCollider](),
		HurtboxColliders:  newStorage[HurtboxCollider](),
		Spawners:          newStorage[Spawner](),
		Timers:            newStorage[Timer](),
		Hitpoints:         newStorage[Hitpoints](),
		ContactDamages:    newStorage[ContactDamage](),
		Abilities:         newStorage[Abilities](),
		FacePositions:     newStorage[FacePosition](),
		Equipments:        newStorage[Equipment](),
		Equippers:         newStorage[Equipper](),
		Deathrattles:      newStorage[Deathrattle](),
		FloatingTexts:     newStorage[FloatingText](),
		ChunkLoaders:      newStorage[ChunkLoader](),
		Persistents:       newStorage[Persistent](),
		Tethers:           newStorage[Tether](),
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

func (x *ECSContainer) SeedRng(seed uint64) {
	x.rngSource.Seed(seed, 0)
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

	var t *Transform
	for _, comp := range comps {
		switch c := comp.(type) {
		case *ChunkLoader:
			return nil
		case *Transform:
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
			err := x.runDeathrattle(e)
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
	// Children are resolved before this entity is deleted, since dropping what they hold needs their ancestors' world
	// transforms
	for _, c := range x.Children(e) {
		linkKind := x.LinkKind(c)

		if linkKind == Link_Attached || (linkKind == Link_Held && dropHeld) {
			if err := x.Detach(c); err != nil {
				log.Printf("error detaching entity %d from removed entity %d: %v", c, e, err)
			}
			continue
		}

		// Parts of a killed entity are removed rather than killed, but still drop what they hold
		x.removeEntity(c, dropHeld && linkKind == Link_Part)
	}

	x.removeChild(x.Parent(e), e)

	for _, s := range x.storages {
		s.deleteEntity(e)
	}

	involvesE := func(p common.EntityPair) bool { return p.A == e || p.B == e }
	x.TickState.AABBCollisions = slices.DeleteFunc(x.TickState.AABBCollisions, involvesE)
	x.TickState.ProximateEntities = slices.DeleteFunc(x.TickState.ProximateEntities, involvesE)
	x.TickState.Collisions = slices.DeleteFunc(x.TickState.Collisions, func(c common.Collision) bool {
		return c.A == e || c.B == e
	})

	for cell, entities := range x.TickState.CollisionGrid {
		if i := slices.Index(entities, e); i != -1 {
			x.TickState.CollisionGrid[cell] = slices.Delete(entities, i, i+1)
		}
	}

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
	members := make(map[common.EntityId][]ComponentDto)
	x.collectComposition(root, members)

	rootIsAttached := x.Parent(root) != -1

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
				if d.Pos, err = x.WorldPos(e); err != nil {
					return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
				}
				if d.Rotation, err = x.WorldRotation(e); err != nil {
					return nil, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
				}
				if d.Scale, err = x.WorldScale(e); err != nil {
					return nil, fmt.Errorf("error getting world scale of entity %d: %v", e, err)
				}
			}
		}
	}

	return members, nil
}

func (x *ECSContainer) collectComposition(e common.EntityId, members map[common.EntityId][]ComponentDto) {
	members[e] = x.GetEntityComponents(e)

	for _, c := range x.Children(e) {
		switch x.LinkKind(c) {
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
