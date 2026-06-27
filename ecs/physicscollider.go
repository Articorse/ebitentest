package ecs

import (
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type PhysicsColliderType uint8

const (
	Collider_Mob PhysicsColliderType = iota
	Collider_Static
	Collider_Trigger
)

type physicsCollider struct {
	baseCollider

	colliderType PhysicsColliderType
}

func (physicsCollider) isComponent() {}

func (x physicsCollider) Copy() physicsCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return physicsCollider{
		colliderType: x.colliderType,
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

func (x *physicsCollider) getBaseCollider() *baseCollider { return &x.baseCollider }

type physicsColliderDto struct {
	ColliderType   PhysicsColliderType
	Enabled        bool
	Shapes         []shapes.ShapeDto
	Center         utils.Vec2
	Aabb           [2]utils.Vec2
	PaddedAabb     [2]utils.Vec2
	CollisionLayer LayerMask
	CollisionMask  LayerMask
}

func (physicsColliderDto) isComponentDto() {}

func (x physicsCollider) ToDto() (physicsColliderDto, error) {
	shapesDto := make([]shapes.ShapeDto, len(x.shapes))
	for i, shape := range x.shapes {
		shapeDto, err := shapes.ShapeToDto(shape)
		if err != nil {
			return physicsColliderDto{}, fmt.Errorf("failed to convert shape to DTO: %w", err)
		}
		shapesDto[i] = shapeDto
	}

	return physicsColliderDto{
		ColliderType:   x.colliderType,
		Enabled:        x.enabled,
		Shapes:         shapesDto,
		Center:         x.center,
		Aabb:           x.aabb,
		PaddedAabb:     x.paddedAabb,
		CollisionLayer: x.collisionLayer,
		CollisionMask:  x.collisionMask,
	}, nil
}

func (x *physicsColliderDto) ToComponent() (*physicsCollider, error) {
	shapesList := make([]shapes.Shape, len(x.Shapes))
	for i, shapeDto := range x.Shapes {
		shape, err := shapes.DtoToShape(shapeDto)
		if err != nil {
			return &physicsCollider{}, fmt.Errorf("failed to convert shape DTO to component: %w", err)
		}
		shapesList[i] = shape
	}

	return &physicsCollider{
		colliderType: x.ColliderType,
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
