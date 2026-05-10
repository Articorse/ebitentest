package ecscommon

type tickState struct {
	Collisions        map[Entity][]Entity
	AABBCollisions    map[Entity][]Entity
	ProximateEntities map[Entity][]Entity
	CollisionGrid     map[CellKey][]Entity
}

type CellKey struct {
	X int
	Y int
}

func NewTickState() *tickState {
	return &tickState{
		Collisions:        make(map[Entity][]Entity),
		AABBCollisions:    make(map[Entity][]Entity),
		ProximateEntities: make(map[Entity][]Entity),
		CollisionGrid:     make(map[CellKey][]Entity),
	}
}
