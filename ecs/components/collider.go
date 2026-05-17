package components

import (
	"ebittest/ecs/components/hitboxes"
	"ebittest/utils"
)

type ColliderType uint8

const (
	Mob ColliderType = iota
	Static
	Trigger
)

type Collider struct {
	Type     ColliderType
	Hitboxes []hitboxes.Hitbox
	center   utils.Vec2
	aabb     [2]utils.Vec2
}

func (Collider) isComponent() {}

func NewColliderComponent(t ColliderType, h []hitboxes.Hitbox) *Collider {
	c := &Collider{
		Type:     t,
		Hitboxes: h,
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

func (x *Collider) GetCenter() utils.Vec2 {
	return x.center
}

func (x *Collider) GetAABB() [2]utils.Vec2 {
	return x.aabb
}
