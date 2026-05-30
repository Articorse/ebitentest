package components

import (
	"ebittest/data"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

type ColliderManager struct{}

func NewColliderComponent(
	cType ColliderType,
	hboxes []hitboxes.Hitbox,
	layers ColliderLayer,
	mask ColliderLayer,
) *Collider {
	c := &Collider{
		colliderType: cType,
		hitboxes:     hboxes,
	}

	c.center = hitboxes.CalculateCenter(hboxes)

	if len(hboxes) == 0 {
		c.aabb = [2]utils.Vec2{
			utils.Vec2{X: 0, Y: 0},
			utils.Vec2{X: 0, Y: 0},
		}
	} else {
		firstAABB := hboxes[0].GetAABB()
		minX, minY := firstAABB[0].X, firstAABB[0].Y
		maxX, maxY := firstAABB[1].X, firstAABB[1].Y

		for _, hitbox := range hboxes {
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

	c.paddedAabb = [2]utils.Vec2{
		utils.Vec2{X: c.aabb[0].X - data.AABBPadding, Y: c.aabb[0].Y - data.AABBPadding},
		utils.Vec2{X: c.aabb[1].X + data.AABBPadding, Y: c.aabb[1].Y + data.AABBPadding},
	}

	c.layers = layers
	c.mask = mask

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

func (*ColliderManager) GetLocalAABB(
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

func (*ColliderManager) GetWorldAABB(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
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
		utils.Vec2{X: colComp.aabb[0].X + worldPos.X, Y: colComp.aabb[0].Y + worldPos.Y},
		utils.Vec2{X: colComp.aabb[1].X + worldPos.X, Y: colComp.aabb[1].Y + worldPos.Y},
	}, nil
}

func (*ColliderManager) GetLocalPaddedAABB(
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

	return collider.paddedAabb, nil
}

func (*ColliderManager) GetWorldPaddedAABB(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
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
		utils.Vec2{X: colComp.paddedAabb[0].X + worldPos.X, Y: colComp.paddedAabb[0].Y + worldPos.Y},
		utils.Vec2{X: colComp.paddedAabb[1].X + worldPos.X, Y: colComp.paddedAabb[1].Y + worldPos.Y},
	}, nil
}

func (*ColliderManager) GetLayers(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) (ColliderLayer, error) {
	collider, ok := colliders[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.layers, nil
}

func (*ColliderManager) GetMask(
	e ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*Collider,
) (ColliderLayer, error) {
	collider, ok := colliders[e]
	if !ok {
		return 0, fmt.Errorf("could not get collider of entity %d", e)
	}

	return collider.mask, nil
}
