package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

type BaseColliderGetter interface {
	getBaseCollider() *baseCollider
}

type IColliderManager interface {
	EntityIds(w *World) []common.EntityId
	HasCollider(e common.EntityId, w *World) bool
	GetWorldPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error)
	GetShapes(e common.EntityId, w *World) ([]shapes.Shape, error)
	GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error)
	GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error)
	GetCenter(e common.EntityId, w *World) (utils.Vec2, error)
	GetLayer(e common.EntityId, w *World) (LayerMask, error)
	GetMask(e common.EntityId, w *World) (LayerMask, error)
}

func newBaseCollider(
	colShapes []shapes.Shape,
	collisionLayer LayerMask,
	collisionMask LayerMask,
) baseCollider {
	c := baseCollider{shapes: colShapes, collisionLayer: collisionLayer, collisionMask: collisionMask}

	c.center = shapes.CalculateCenter(colShapes)

	if len(colShapes) == 0 {
		c.aabb = [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
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

		c.aabb = [2]utils.Vec2{
			utils.Vec2{X: minX, Y: minY},
			utils.Vec2{X: maxX, Y: maxY},
		}
	}

	c.paddedAabb = [2]utils.Vec2{
		utils.Vec2{X: c.aabb[0].X - data.AABBPadding, Y: c.aabb[0].Y - data.AABBPadding},
		utils.Vec2{X: c.aabb[1].X + data.AABBPadding, Y: c.aabb[1].Y + data.AABBPadding},
	}

	return c
}

type BaseColliderManager[T BaseColliderGetter] struct{}

func getCollider[T BaseColliderGetter](e common.EntityId, world *World) (T, error) {
	var t T
	switch BaseColliderGetter(t).(type) {
	case *hitboxCollider:
		c, err := world.HitboxColliders.getComponent(e)
		if err != nil {
			return *new(T), fmt.Errorf("could not get hitbox collider of entity %d: %v", e, err)
		}
		return any(c).(T), nil
	case *hurtboxCollider:
		c, err := world.HurtboxColliders.getComponent(e)
		if err != nil {
			return *new(T), fmt.Errorf("could not get hurtbox collider of entity %d: %v", e, err)
		}
		return any(c).(T), nil
	case *physicsCollider:
		c, err := world.PhysicsColliders.getComponent(e)
		if err != nil {
			return *new(T), fmt.Errorf("could not get physics collider of entity %d: %v", e, err)
		}
		return any(c).(T), nil
	case *platformCollider:
		c, err := world.PlatformColliders.getComponent(e)
		if err != nil {
			return *new(T), fmt.Errorf("could not get platform collider of entity %d: %v", e, err)
		}
		return any(c).(T), nil
	default:
		return *new(T), fmt.Errorf("unsupported collider type %T", t)
	}
}

func (BaseColliderManager[T]) GetShapes(
	e common.EntityId,
	world *World,
) ([]shapes.Shape, error) {
	collider, err := getCollider[T](e, world)
	if err != nil {
		return nil, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}
	return collider.getBaseCollider().shapes, nil
}

func (BaseColliderManager[T]) GetCenter(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	collider, err := getCollider[T](e, world)
	if err != nil {
		return utils.Vec2{X: 0, Y: 0}, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.getBaseCollider().center, nil
}

func (BaseColliderManager[T]) GetLocalAABB(
	e common.EntityId,
	world *World,
) ([2]utils.Vec2, error) {
	collider, err := getCollider[T](e, world)
	if err != nil {
		return [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.getBaseCollider().aabb, nil
}

func (BaseColliderManager[T]) GetWorldAABB(
	e common.EntityId,
	world *World,
) ([2]utils.Vec2, error) {
	tm := transformManager{}

	colComp, err := getCollider[T](e, world)
	if err != nil {
		return [2]utils.Vec2{}, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	worldPos, err := tm.GetWorldPos(e, world)
	if err != nil {
		return [2]utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	return [2]utils.Vec2{
		utils.Vec2{X: colComp.getBaseCollider().aabb[0].X + worldPos.X, Y: colComp.getBaseCollider().aabb[0].Y + worldPos.Y},
		utils.Vec2{X: colComp.getBaseCollider().aabb[1].X + worldPos.X, Y: colComp.getBaseCollider().aabb[1].Y + worldPos.Y},
	}, nil
}

func (BaseColliderManager[T]) GetLocalPaddedAABB(
	e common.EntityId,
	world *World,
) ([2]utils.Vec2, error) {
	collider, err := getCollider[T](e, world)
	if err != nil {
		return [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return collider.getBaseCollider().paddedAabb, nil
}

func (BaseColliderManager[T]) GetWorldPaddedAABB(
	e common.EntityId,
	world *World,
) ([2]utils.Vec2, error) {
	tm := transformManager{}

	colComp, err := getCollider[T](e, world)
	if err != nil {
		return [2]utils.Vec2{}, fmt.Errorf("could not get transform of entity %d: %v", e, err)
	}

	worldPos, err := tm.GetWorldPos(e, world)
	if err != nil {
		return [2]utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	return [2]utils.Vec2{
		utils.Vec2{X: colComp.getBaseCollider().paddedAabb[0].X + worldPos.X, Y: colComp.getBaseCollider().paddedAabb[0].Y + worldPos.Y},
		utils.Vec2{X: colComp.getBaseCollider().paddedAabb[1].X + worldPos.X, Y: colComp.getBaseCollider().paddedAabb[1].Y + worldPos.Y},
	}, nil
}

func (BaseColliderManager[T]) GetLayer(
	e common.EntityId,
	world *World,
) (LayerMask, error) {
	colComp, err := getCollider[T](e, world)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return colComp.getBaseCollider().collisionLayer, nil
}

func (BaseColliderManager[T]) GetMask(
	e common.EntityId,
	world *World,
) (LayerMask, error) {
	colComp, err := getCollider[T](e, world)
	if err != nil {
		return 0, fmt.Errorf("could not get collider of entity %d: %v", e, err)
	}

	return colComp.getBaseCollider().collisionMask, nil
}
