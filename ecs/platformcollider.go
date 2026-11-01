package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type PlatformCollider struct {
	baseCollider
}

func NewPlatformColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes []shapes.Shape,
) *PlatformCollider {
	return &PlatformCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}
func (x PlatformCollider) Copy() PlatformCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return PlatformCollider{
		baseCollider: baseCollider{
			enabled:        x.enabled,
			shapes:         colShapesCopy,
			center:         x.center,
			aabb:           x.aabb,
			paddedAabb:     x.paddedAabb,
			collisionLayer: x.collisionLayer,
			collisionMask:  x.collisionMask,
		},
	}
}

func (x PlatformCollider) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.PlatformColliders.addComponent(e, x.Copy())
}

type platformColliderDto struct {
	Enabled        bool
	Shapes         []shapes.ShapeDto
	Center         utils.Vec2f
	Aabb           [2]utils.Vec2f
	PaddedAabb     [2]utils.Vec2f
	CollisionLayer LayerMask
	CollisionMask  LayerMask
}

func (x PlatformCollider) toDto() (ComponentDto, error) {
	shapesDto := make([]shapes.ShapeDto, len(x.shapes))
	for i, shape := range x.shapes {
		shapeDto, err := shapes.ShapeToDto(shape)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape to DTO: %w", err)
		}
		shapesDto[i] = shapeDto
	}

	return &platformColliderDto{
		Enabled:        x.enabled,
		Shapes:         shapesDto,
		Center:         x.center,
		Aabb:           x.aabb,
		PaddedAabb:     x.paddedAabb,
		CollisionLayer: x.collisionLayer,
		CollisionMask:  x.collisionMask,
	}, nil
}

func (x *platformColliderDto) toComponent() (Component, error) {
	shapesList := make([]shapes.Shape, len(x.Shapes))
	for i, shapeDto := range x.Shapes {
		shape, err := shapes.DtoToShape(shapeDto)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape DTO to component: %w", err)
		}
		shapesList[i] = shape
	}

	return &PlatformCollider{
		baseCollider: baseCollider{
			enabled:        x.Enabled,
			shapes:         shapesList,
			center:         x.Center,
			aabb:           x.Aabb,
			paddedAabb:     x.PaddedAabb,
			collisionLayer: x.CollisionLayer,
			collisionMask:  x.CollisionMask,
		},
	}, nil
}
