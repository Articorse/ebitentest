package ecscommon

type TickState struct {
	Collisions        map[EntityId][]EntityId
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
		Collisions:        make(map[EntityId][]EntityId),
		AABBCollisions:    make(map[EntityId][]EntityId),
		ProximateEntities: make(map[EntityId][]EntityId),
		CollisionGrid:     make(map[CellKey][]EntityId),
	}
}
