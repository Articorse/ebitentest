package ecs

import (
	"ebittest/data"
	"ebittest/ecs/collidershapes"
	"ebittest/ecs/common"
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
	GetShapes(e common.EntityId, w *World) ([]collidershapes.Shape, error)
	GetLocalAABB(e common.EntityId, w *World) ([2]utils.Vec2, error)
	GetLocalPaddedAABB(e common.EntityId, w *World) ([2]utils.Vec2, error)
	GetCenter(e common.EntityId, w *World) (utils.Vec2, error)
}

func newBaseCollider(shapes []collidershapes.Shape) baseCollider {
	c := baseCollider{shapes: shapes}

	c.center = collidershapes.CalculateCenter(shapes)

	if len(shapes) == 0 {
		c.aabb = [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}
	} else {
		firstAABB := shapes[0].GetAABB()
		minX, minY := firstAABB[0].X, firstAABB[0].Y
		maxX, maxY := firstAABB[1].X, firstAABB[1].Y

		for _, shape := range shapes {
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

func (BaseColliderManager[T]) GetShapes(
	e common.EntityId,
	colliders map[common.EntityId]T,
) ([]collidershapes.Shape, error) {
	collider, ok := colliders[e]
	if !ok {
		return nil, fmt.Errorf("could not get collider of entity %d", e)
	}
	return collider.getBaseCollider().shapes, nil
}

func (BaseColliderManager[T]) GetCenter(
	e common.EntityId,
	colliders map[common.EntityId]T,
) (utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return utils.Vec2{X: 0, Y: 0}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.getBaseCollider().center, nil
}

func (BaseColliderManager[T]) GetLocalAABB(
	e common.EntityId,
	colliders map[common.EntityId]T,
) ([2]utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.getBaseCollider().aabb, nil
}

func (BaseColliderManager[T]) GetWorldAABB(
	e common.EntityId,
	colliders map[common.EntityId]T,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) ([2]utils.Vec2, error) {
	tm := TransformManager{}

	colComp, ok := colliders[e]
	if !ok {
		return [2]utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	worldPos, err := tm.GetWorldPos(e, transforms, parents)
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
	colliders map[common.EntityId]T,
) ([2]utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.getBaseCollider().paddedAabb, nil
}

func (BaseColliderManager[T]) GetWorldPaddedAABB(
	e common.EntityId,
	colliders map[common.EntityId]T,
	transforms map[common.EntityId]*transform,
	parents map[common.EntityId]*parent,
) ([2]utils.Vec2, error) {
	tm := TransformManager{}

	colComp, ok := colliders[e]
	if !ok {
		return [2]utils.Vec2{}, fmt.Errorf("could not get transform of entity %d", e)
	}

	worldPos, err := tm.GetWorldPos(e, transforms, parents)
	if err != nil {
		return [2]utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	return [2]utils.Vec2{
		utils.Vec2{X: colComp.getBaseCollider().paddedAabb[0].X + worldPos.X, Y: colComp.getBaseCollider().paddedAabb[0].Y + worldPos.Y},
		utils.Vec2{X: colComp.getBaseCollider().paddedAabb[1].X + worldPos.X, Y: colComp.getBaseCollider().paddedAabb[1].Y + worldPos.Y},
	}, nil
}
