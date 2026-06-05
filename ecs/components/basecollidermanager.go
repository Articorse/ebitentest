package components

import (
	"ebittest/data"
	"ebittest/ecs/components/collidershapes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

type baseColliderGetter interface {
	getBaseCollider() *BaseColliderComponent
}

func newBaseCollider(shapes []collidershapes.Shape) BaseColliderComponent {
	c := BaseColliderComponent{shapes: shapes}

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

type BaseColliderManager[T baseColliderGetter] struct{}

func (BaseColliderManager[T]) GetShapes(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
) ([]collidershapes.Shape, error) {
	collider, ok := colliders[e]
	if !ok {
		return nil, fmt.Errorf("could not get collider of entity %d", e)
	}
	return collider.getBaseCollider().shapes, nil
}

func (BaseColliderManager[T]) GetCenter(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
) (utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return utils.Vec2{X: 0, Y: 0}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.getBaseCollider().center, nil
}

func (BaseColliderManager[T]) GetLocalAABB(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
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
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
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
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]T,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
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
