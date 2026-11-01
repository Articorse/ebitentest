package ecs

import (
	"ebittest/ecs/common"
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

type PhysicsCollider struct {
	baseCollider

	colliderType PhysicsColliderType
}

func NewPhysicsColliderComponent(
	collisionLayer LayerMask,
	collisionMask LayerMask,
	cType PhysicsColliderType,
	shapes ...shapes.Shape,
) *PhysicsCollider {
	return &PhysicsCollider{
		colliderType: cType,
		baseCollider: newBaseCollider(shapes, collisionLayer, collisionMask),
	}
}
func (x *PhysicsCollider) ColliderType() PhysicsColliderType {
	return x.colliderType
}

func (x PhysicsCollider) Copy() PhysicsCollider {
	colShapesCopy := make([]shapes.Shape, len(x.shapes))
	for i, shape := range x.shapes {
		colShapesCopy[i] = shape.Copy()
	}

	return PhysicsCollider{
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

func (x PhysicsCollider) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.PhysicsColliders.addComponent(e, x.Copy())
}

type physicsColliderDto struct {
	ColliderType   PhysicsColliderType
	Enabled        bool
	Shapes         []shapes.ShapeDto
	Center         utils.Vec2f
	Aabb           [2]utils.Vec2f
	PaddedAabb     [2]utils.Vec2f
	CollisionLayer LayerMask
	CollisionMask  LayerMask
}

func (x PhysicsCollider) toDto() (ComponentDto, error) {
	shapesDto := make([]shapes.ShapeDto, len(x.shapes))
	for i, shape := range x.shapes {
		shapeDto, err := shapes.ShapeToDto(shape)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape to DTO: %w", err)
		}
		shapesDto[i] = shapeDto
	}

	return &physicsColliderDto{
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

func (x *physicsColliderDto) toComponent() (Component, error) {
	shapesList := make([]shapes.Shape, len(x.Shapes))
	for i, shapeDto := range x.Shapes {
		shape, err := shapes.DtoToShape(shapeDto)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shape DTO to component: %w", err)
		}
		shapesList[i] = shape
	}

	return &PhysicsCollider{
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
