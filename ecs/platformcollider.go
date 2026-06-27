package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type platformCollider struct {
	baseCollider
}

func (x *platformCollider) getBaseCollider() *baseCollider { return &x.baseCollider }

func (platformCollider) isComponent() {}

func (x platformCollider) Copy() platformCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return platformCollider{
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

type platformColliderDto struct {
	Enabled        bool
	Shapes         []shapes.ShapeDto
	Center         utils.Vec2
	Aabb           [2]utils.Vec2
	PaddedAabb     [2]utils.Vec2
	CollisionLayer LayerMask
	CollisionMask  LayerMask
}

func (platformColliderDto) isComponentDto() {}

func (x platformCollider) ToDto() (platformColliderDto, error) {
	shapesDto := make([]shapes.ShapeDto, len(x.shapes))
	for i, shape := range x.shapes {
		shapeDto, err := shapes.ShapeToDto(shape)
		if err != nil {
			return platformColliderDto{}, fmt.Errorf("failed to convert shape to DTO: %w", err)
		}
		shapesDto[i] = shapeDto
	}

	return platformColliderDto{
		Enabled:        x.enabled,
		Shapes:         shapesDto,
		Center:         x.center,
		Aabb:           x.aabb,
		PaddedAabb:     x.paddedAabb,
		CollisionLayer: x.collisionLayer,
		CollisionMask:  x.collisionMask,
	}, nil
}

func (x *platformColliderDto) ToComponent() (*platformCollider, error) {
	shapesList := make([]shapes.Shape, len(x.Shapes))
	for i, shapeDto := range x.Shapes {
		shape, err := shapes.DtoToShape(shapeDto)
		if err != nil {
			return &platformCollider{}, fmt.Errorf("failed to convert shape DTO to component: %w", err)
		}
		shapesList[i] = shape
	}

	return &platformCollider{
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
