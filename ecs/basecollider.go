package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type LayerMask uint16

const (
	Layer_Player             = LayerMask(0b1000000000000000)
	Layer_Enemy              = LayerMask(0b0100000000000000)
	Layer_FriendlyProjectile = LayerMask(0b0010000000000000)
	Layer_EnemyProjectile    = LayerMask(0b0001000000000000)
	Layer_Terrain            = LayerMask(0b0000100000000000)
	Layer_Platform           = LayerMask(0b0000010000000000)
	Layer_Ground = LayerMask(0b0000001000000000)
	Layer_Feature = LayerMask(0b0000000100000000)
)

type baseCollider struct {
	enabled        bool
	shapes         []shapes.Shape
	center         utils.Vec2f
	collisionLayer LayerMask
	collisionMask  LayerMask
}

func newBaseCollider(
	colShapes []shapes.Shape,
	collisionLayer LayerMask,
	collisionMask LayerMask,
) baseCollider {
	c := baseCollider{enabled: true, shapes: colShapes, collisionLayer: collisionLayer, collisionMask: collisionMask}

	c.center = shapes.CalculateCenter(colShapes)

	return c
}

func (x *baseCollider) Enabled() bool {
	return x.enabled
}

func (x *baseCollider) SetEnabled(enabled bool) {
	x.enabled = enabled
}

func (x *baseCollider) Shapes() []shapes.Shape {
	return x.shapes
}

func (x *baseCollider) Center() utils.Vec2f {
	return x.center
}

func (x *baseCollider) Layer() LayerMask {
	return x.collisionLayer
}

func (x *baseCollider) Mask() LayerMask {
	return x.collisionMask
}

type Collider interface {
	Enabled() bool
	SetEnabled(enabled bool)
	Shapes() []shapes.Shape
	Center() utils.Vec2f
	Layer() LayerMask
	Mask() LayerMask
}

type ColliderSet interface {
	GetEntities() []common.EntityId
	Collider(e common.EntityId) (Collider, bool)
}

type colliderSet[T Component, PT interface {
	*T
	Collider
}] struct {
	storage *Storage[T]
}

func Colliders[T Component, PT interface {
	*T
	Collider
}](storage *Storage[T]) ColliderSet {
	return colliderSet[T, PT]{storage: storage}
}

func (x colliderSet[T, PT]) GetEntities() []common.EntityId {
	return x.storage.GetEntities()
}

func (x colliderSet[T, PT]) Collider(e common.EntityId) (Collider, bool) {
	c, ok := x.storage.Get(e)
	if !ok {
		return nil, false
	}

	return PT(c), true
}

// Turned and scaled with the entity, but still around its position, as the shapes of its colliders are
func (x *ECSContainer) TransformShape(e common.EntityId, s shapes.Shape) (shapes.Shape, error) {
	rotation, err := x.WorldRotation(e)
	if err != nil {
		return nil, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}
	scale, err := x.WorldScale(e)
	if err != nil {
		return nil, fmt.Errorf("error getting world scale of entity %d: %v", e, err)
	}

	return shapes.Transform(s, rotation, scale), nil
}

// Around the collider's shapes as they're turned and scaled with the entity
func (x *ECSContainer) ColliderWorldAABB(e common.EntityId, col Collider) ([2]utils.Vec2f, error) {
	worldPos, err := x.WorldPos(e)
	if err != nil {
		return [2]utils.Vec2f{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	local := make([]shapes.Shape, len(col.Shapes()))
	for i, s := range col.Shapes() {
		if local[i], err = x.TransformShape(e, s); err != nil {
			return [2]utils.Vec2f{}, err
		}
	}

	aabb := shapes.BoundingAABB(local)
	return [2]utils.Vec2f{aabb[0].Add(worldPos), aabb[1].Add(worldPos)}, nil
}

// Padded by data.AABBPadding, so that the broad phase also catches shapes that are about to touch
func (x *ECSContainer) ColliderPaddedWorldAABB(e common.EntityId, col Collider) ([2]utils.Vec2f, error) {
	aabb, err := x.ColliderWorldAABB(e, col)
	padding := utils.Vec2f{X: data.AABBPadding, Y: data.AABBPadding}
	return [2]utils.Vec2f{aabb[0].Subtract(padding), aabb[1].Add(padding)}, err
}
