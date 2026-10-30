package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type SpawnerType uint8

const (
	SpawnerType_Point SpawnerType = iota
	SpawnerType_Inside
	SpawnerType_Perimeter
)

type spawner struct {
	offset      utils.Vec2f
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

type spawnerDto struct {
	Offset      utils.Vec2f
	SpawnerType SpawnerType
	Shape       *shapes.ShapeDto
	Components  []ComponentDto
}

func (spawnerDto) isComponentDto() {}

func (x spawner) ToDto() (spawnerDto, error) {
	componentsDto := make([]ComponentDto, len(x.components))
	for i, c := range x.components {
		dto, err := ComponentToDto(c)
		if err != nil {
			return spawnerDto{}, fmt.Errorf("failed to convert component to dto: %w", err)
		}
		componentsDto[i] = dto
	}

	var shapeDto *shapes.ShapeDto
	if x.shape != nil {
		dto, err := shapes.ShapeToDto(x.shape)
		if err != nil {
			return spawnerDto{}, fmt.Errorf("failed to convert shape to dto: %w", err)
		}
		shapeDto = &dto
	}

	return spawnerDto{
		Offset:      x.offset,
		SpawnerType: x.spawnerType,
		Shape:       shapeDto,
		Components:  componentsDto,
	}, nil
}

func (x *spawnerDto) ToComponent() (*spawner, error) {
	components := make([]Component, len(x.Components))
	for i, dto := range x.Components {
		comp, err := DtoToComponent(dto)
		if err != nil {
			return &spawner{}, fmt.Errorf("failed to convert component dto to component: %w", err)
		}
		components[i] = comp
	}

	var shape shapes.Shape
	if x.Shape != nil {
		var err error
		shape, err = shapes.DtoToShape(*x.Shape)
		if err != nil {
			return &spawner{}, fmt.Errorf("failed to convert shape dto to shape: %w", err)
		}
	}

	return &spawner{
		offset:      x.Offset,
		spawnerType: x.SpawnerType,
		shape:       shape,
		components:  components,
	}, nil
}
