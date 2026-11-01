package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
	"math"
	"slices"
)

type SpawnerType uint8

const (
	SpawnerType_Point SpawnerType = iota
	SpawnerType_Inside
	SpawnerType_Perimeter
)

type Spawner struct {
	offset      utils.Vec2f
	spawnerType SpawnerType
	shape       shapes.Shape
	components  []Component
}

// TODO: Might require a deep copy
func NewSpawnerComponent(
	offset utils.Vec2f,
	sType SpawnerType,
	shape shapes.Shape,
	components ...Component,
) (*Spawner, error) {
	switch sType {
	case SpawnerType_Point:
		if shape != nil {
			return nil, fmt.Errorf("shape must be nil for point spawner type")
		}
	case SpawnerType_Inside, SpawnerType_Perimeter:
		if shape == nil {
			return nil, fmt.Errorf("shape must be non-nil for inside and perimeter spawner types")
		}
	default:
		return nil, fmt.Errorf("invalid spawner type: %d", sType)
	}

	return &Spawner{offset: offset, spawnerType: sType, shape: shape, components: components}, nil
}

func (x *Spawner) Offset() utils.Vec2f {
	return x.offset
}

func (x *Spawner) Type() SpawnerType {
	return x.spawnerType
}

func (x *Spawner) Shape() shapes.Shape {
	return x.shape
}

// Shared by every spawn, so they must not be modified
func (x *Spawner) Components() []Component {
	return slices.Clone(x.components)
}

func (x *ECSContainer) Spawn(spawnerEntity common.EntityId) (common.EntityId, error) {
	s, ok := x.Spawners.Get(spawnerEntity)
	if !ok {
		return -1, fmt.Errorf("entity %d has no spawner", spawnerEntity)
	}

	// If there is no Transform, then both of these are zero, which is fine if spawning based on camera or other entity
	worldPos, _ := x.WorldPos(spawnerEntity)
	worldRot, _ := x.WorldRotation(spawnerEntity)

	var finalOffset utils.Vec2f
	switch s.spawnerType {
	case SpawnerType_Point:
		cos := math.Cos(worldRot)
		sin := math.Sin(worldRot)

		rotatedOffset := utils.Vec2f{
			X: (s.offset.X*cos - s.offset.Y*sin),
			Y: (s.offset.X*sin + s.offset.Y*cos),
		}

		finalOffset = rotatedOffset
	case SpawnerType_Inside:
		finalOffset = s.shape.GetRandomPoint(x.Rng).Add(s.offset)
	case SpawnerType_Perimeter:
		finalOffset = s.shape.GetRandomPointAroundShape(x.Rng).Add(s.offset)
	}

	// Placed before spawning, since the spawn position decides whether spawning is allowed.
	// The spawner's components are shared by every spawn, so its transform is replaced rather than modified.
	comps := slices.Clone(s.components)
	tIdx := slices.IndexFunc(comps, func(c Component) bool {
		_, ok := c.(*Transform)
		return ok
	})
	if tIdx < 0 {
		return -1, fmt.Errorf("components of spawner entity %d have no transform", spawnerEntity)
	}
	t := comps[tIdx].(*Transform).Copy()
	t.pos = worldPos.Add(finalOffset)
	t.rotation = worldRot
	comps[tIdx] = &t

	newEntity, err := x.AddNewEntity(comps...)
	if err != nil {
		return -1, fmt.Errorf("error spawning entity of spawner entity %d: %v", spawnerEntity, err)
	}

	return newEntity, nil
}

func (x Spawner) Copy() Spawner {
	componentsCopy := make([]Component, len(x.components))
	copy(componentsCopy, x.components)
	var shape shapes.Shape
	if x.shape != nil {
		shape = x.shape.Copy()
	}

	return Spawner{
		offset:      x.offset,
		spawnerType: x.spawnerType,
		shape:       shape,
		components:  componentsCopy,
	}
}

func (x Spawner) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Spawners.addComponent(e, x.Copy())
}

type spawnerDto struct {
	Offset      utils.Vec2f
	SpawnerType SpawnerType
	Shape       *shapes.ShapeDto
	Components  []ComponentDto
}

func (x Spawner) toDto() (ComponentDto, error) {
	componentsDto := make([]ComponentDto, len(x.components))
	for i, c := range x.components {
		dto, err := ComponentToDto(c)
		if err != nil {
			return nil, fmt.Errorf("failed to convert component to dto: %w", err)
		}
		componentsDto[i] = dto
	}

	var shapeDto *shapes.ShapeDto
	if x.shape != nil {
		dto, err := shapes.ShapeToDto(x.shape)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape to dto: %w", err)
		}
		shapeDto = &dto
	}

	return &spawnerDto{
		Offset:      x.offset,
		SpawnerType: x.spawnerType,
		Shape:       shapeDto,
		Components:  componentsDto,
	}, nil
}

func (x *spawnerDto) toComponent() (Component, error) {
	components := make([]Component, len(x.Components))
	for i, dto := range x.Components {
		comp, err := DtoToComponent(dto)
		if err != nil {
			return nil, fmt.Errorf("failed to convert component dto to component: %w", err)
		}
		components[i] = comp
	}

	var shape shapes.Shape
	if x.Shape != nil {
		var err error
		shape, err = shapes.DtoToShape(*x.Shape)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape dto to shape: %w", err)
		}
	}

	return &Spawner{
		offset:      x.Offset,
		spawnerType: x.SpawnerType,
		shape:       shape,
		components:  components,
	}, nil
}
