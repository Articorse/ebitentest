package components

import (
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

type ColliderManager struct{}

func NewColliderComponent(t ColliderType, h []hitboxes.Hitbox) *Collider {
	c := &Collider{
		colliderType: t,
		hitboxes:     h,
	}

	c.center = hitboxes.CalculateCenter(h)

	if len(h) == 0 {
		c.aabb = [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}
	} else {
		firstAABB := h[0].GetAABB()
		minX, minY := firstAABB[0].X, firstAABB[0].Y
		maxX, maxY := firstAABB[1].X, firstAABB[1].Y

		for _, hitbox := range h {
			aabb := hitbox.GetAABB()
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

	return c
}

func (*ColliderManager) GetColliderType(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) (ColliderType, error) {
	collider, ok := colliders[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.colliderType, nil
}

func (*ColliderManager) GetHitboxes(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) ([]hitboxes.Hitbox, error) {
	collider, ok := colliders[e]
	if !ok {
		return nil, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.hitboxes, nil
}

func (*ColliderManager) GetCenter(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) (utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return utils.Vec2{X: 0, Y: 0}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.center, nil
}

func (*ColliderManager) GetAABB(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) ([2]utils.Vec2, error) {
	collider, ok := colliders[e]
	if !ok {
		return [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.aabb, nil
}
