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
)

type baseCollider struct {
	enabled        bool
	shapes         []shapes.Shape
	center         utils.Vec2f
	aabb           [2]utils.Vec2f
	paddedAabb     [2]utils.Vec2f
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

	if len(colShapes) == 0 {
		c.aabb = [2]utils.Vec2f{
			utils.Vec2f{X: 0, Y: 0},
			utils.Vec2f{X: 0, Y: 0},
		}
	} else {
		firstAABB := colShapes[0].GetAABB()
		minX, minY := firstAABB[0].X, firstAABB[0].Y
		maxX, maxY := firstAABB[1].X, firstAABB[1].Y

		for _, shape := range colShapes {
			aabb := shape.GetAABB()
			if aabb[0].X < minX {
				minX = aabb[0].X
			}
			if aabb[0].Y < minY {
				minY = aabb[0].Y
			}
			if aabb[1].X > maxX {
				maxX = aabb[1].X
			}
			if aabb[1].Y > maxY {
				maxY = aabb[1].Y
			}
		}

		c.aabb = [2]utils.Vec2f{
			utils.Vec2f{X: minX, Y: minY},
			utils.Vec2f{X: maxX, Y: maxY},
		}
	}

	c.paddedAabb = [2]utils.Vec2f{
		utils.Vec2f{X: c.aabb[0].X - data.AABBPadding, Y: c.aabb[0].Y - data.AABBPadding},
		utils.Vec2f{X: c.aabb[1].X + data.AABBPadding, Y: c.aabb[1].Y + data.AABBPadding},
	}

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

func (x *baseCollider) LocalAABB() [2]utils.Vec2f {
	return x.aabb
}

// Padded by data.AABBPadding, so that the broad phase also catches shapes that are about to touch
func (x *baseCollider) LocalPaddedAABB() [2]utils.Vec2f {
	return x.paddedAabb
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
	LocalAABB() [2]utils.Vec2f
	LocalPaddedAABB() [2]utils.Vec2f
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

func (x *ECSContainer) WorldAABB(e common.EntityId, localAABB [2]utils.Vec2f) ([2]utils.Vec2f, error) {
	worldPos, err := x.WorldPos(e)
	if err != nil {
		return [2]utils.Vec2f{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	return [2]utils.Vec2f{localAABB[0].Add(worldPos), localAABB[1].Add(worldPos)}, nil
}
