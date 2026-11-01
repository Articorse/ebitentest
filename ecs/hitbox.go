package ecs

import (
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type HitboxCollider struct {
	baseCollider
}

func NewHitboxColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	shapes ...shapes.Shape,
) *HitboxCollider {
	return &HitboxCollider{baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask)}
}
func (x HitboxCollider) Copy() HitboxCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return HitboxCollider{
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

func (x HitboxCollider) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.HitboxColliders.addComponent(e, x.Copy())
}

type hitboxColliderDto struct {
	Enabled        bool
	Shapes         []shapes.ShapeDto
	Center         utils.Vec2f
	Aabb           [2]utils.Vec2f
	PaddedAabb     [2]utils.Vec2f
	CollisionLayer LayerMask
	CollisionMask  LayerMask
}

func (x HitboxCollider) toDto() (ComponentDto, error) {
	shapesDto := make([]shapes.ShapeDto, len(x.shapes))
	for i, shape := range x.shapes {
		shapeDto, err := shapes.ShapeToDto(shape)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape to DTO: %w", err)
		}
		shapesDto[i] = shapeDto
	}

	return &hitboxColliderDto{
		Enabled:        x.enabled,
		Shapes:         shapesDto,
		Center:         x.center,
		Aabb:           x.aabb,
		PaddedAabb:     x.paddedAabb,
		CollisionLayer: x.collisionLayer,
		CollisionMask:  x.collisionMask,
	}, nil
}

func (x *hitboxColliderDto) toComponent() (Component, error) {
	shapesList := make([]shapes.Shape, len(x.Shapes))
	for i, shapeDto := range x.Shapes {
		shape, err := shapes.DtoToShape(shapeDto)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape DTO to component: %w", err)
		}
		shapesList[i] = shape
	}

	return &HitboxCollider{
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
