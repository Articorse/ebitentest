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
	components  []component
}

func (spawner) isComponent() {}

func (x spawner) Copy() spawner {
	componentsCopy := make([]component, len(x.components))
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
