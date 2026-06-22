package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
)

type SpawnerType uint8

const (
	SpawnerType_Point SpawnerType = iota
	SpawnerType_Inside
	SpawnerType_Perimeter
)

type spawner struct {
	offset      utils.Vec2
	spawnerType SpawnerType
	shape       shapes.Shape
	components  []Component
}

func (spawner) isComponent() {}

// TODO: Might require a deep copy
func (x spawner) Copy() spawner {
	componentsCopy := make([]Component, len(x.components))
	copy(componentsCopy, x.components)
	var shape shapes.Shape
	if x.shape != nil {
		shape = x.shape.Copy()
	}

	return spawner{
		offset:      x.offset,
		spawnerType: x.spawnerType,
		shape:       shape,
		components:  componentsCopy,
	}
}
