package common

type TickState struct {
	Collisions        map[EntityId]map[EntityId]Collision
	AABBCollisions    map[EntityId][]EntityId
	ProximateEntities map[EntityId][]EntityId
	CollisionGrid     map[CellKey][]EntityId
}

type CellKey struct {
	X int
	Y int
}

func NewTickState() *TickState {
	return &TickState{
		Collisions:        make(map[EntityId]map[EntityId]Collision),
		AABBCollisions:    make(map[EntityId][]EntityId),
		ProximateEntities: make(map[EntityId][]EntityId),
		CollisionGrid:     make(map[CellKey][]EntityId),
	}
}
