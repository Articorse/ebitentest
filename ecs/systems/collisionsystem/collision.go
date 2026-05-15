package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func GetCollisions(
	potentialCollisions map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)

	for eA, colEntities := range potentialCollisions {
		for _, eB := range colEntities {
			colA, ok := colliders[eA]
			if !ok {
				return nil, fmt.Errorf("colliding entity has no collider: ", eA)
			}

			colB, ok := colliders[eB]
			if !ok {
				return nil, fmt.Errorf("colliding entity has no collider: ", eB)
			}

			traA, ok := transforms[eA]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eB,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}
			}

			if eA == eB {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			var worldA []utils.Vec2
			for _, v := range colA.Vertices {
				worldA = append(worldA, utils.Vec2{X: traA.Pos.X + v.X, Y: traA.Pos.Y + v.Y})
			}
			var worldB []utils.Vec2
			for _, v := range colB.Vertices {
				worldB = append(worldB, utils.Vec2{X: traB.Pos.X + v.X, Y: traB.Pos.Y + v.Y})
			}

			intersectionFound := false
			for i := 0; i < len(worldA)-1; i++ {
				if intersectionFound {
					break
				}
				for j := 0; j < len(worldB)-1; j++ {
					if utils.SegmentsIntersect(worldA[i], worldA[i+1], worldB[j], worldB[j+1]) {
						intersectionFound = true
						break
					}
				}
			}

			if intersectionFound {
				v, ok := collisions[eA]
				if !ok {
					collisions[eA] = []ecscommon.EntityId{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	proximateEntities map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)

	for eA, colEntities := range proximateEntities {
		for _, eB := range colEntities {
			if eA == eB {
				continue
			}

			colA, ok := colliders[eA]
			if !ok {
				return nil, fmt.Errorf("colliding entity has no collider: ", eA)
			}

			colB, ok := colliders[eB]
			if !ok {
				return nil, fmt.Errorf("colliding entity has no collider: ", eB)
			}

			traA, ok := transforms[eA]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eB,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if slices.Contains(collidedEntities, eA) {
					continue
				}
			}

			a := []utils.Vec2{
				utils.Vec2{X: traA.Pos.X + colA.AABB[0].X, Y: traA.Pos.Y + colA.AABB[0].Y},
				utils.Vec2{X: traA.Pos.X + colA.AABB[1].X, Y: traA.Pos.Y + colA.AABB[1].Y},
			}
			b := []utils.Vec2{
				utils.Vec2{X: traB.Pos.X + colB.AABB[0].X, Y: traB.Pos.Y + colB.AABB[0].Y},
				utils.Vec2{X: traB.Pos.X + colB.AABB[1].X, Y: traB.Pos.Y + colB.AABB[1].Y},
			}

			if detectAABBCollision(a, b) {
				v, ok := collisions[eA]
				if !ok {
					collisions[eA] = []ecscommon.EntityId{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}

func detectAABBCollision(a, b []utils.Vec2) bool {
	minAx := a[0].X
	minAy := a[0].Y
	maxAx := a[0].X
	maxAy := a[0].Y
	for _, v := range a {
		if v.X < minAx {
			minAx = v.X
		}
		if v.X > maxAx {
			maxAx = v.X
		}
		if v.Y < minAy {
			minAy = v.Y
		}
		if v.Y > maxAy {
			maxAy = v.Y
		}
	}

	minBx := b[0].X
	minBy := b[0].Y
	maxBx := b[0].X
	maxBy := b[0].Y
	for _, v := range b {
		if v.X < minBx {
			minBx = v.X
		}
		if v.X > maxBx {
			maxBx = v.X
		}
		if v.Y < minBy {
			minBy = v.Y
		}
		if v.Y > maxBy {
			maxBy = v.Y
		}
	}

	return minAx <= maxBx && maxAx >= minBx && minAy <= maxBy && maxAy >= minBy
}

func DrawColliders(
	screen *ebiten.Image,
	camera utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	collisions map[ecscommon.EntityId][]ecscommon.EntityId,
) error {
	for e, col := range colliders {
		tra, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Collider",
				MissingComponent: "Transform",
			}
		}

		lineColor := data.Debug_ColliderColor

		isColliding := false
		if _, ok := collisions[e]; ok {
			isColliding = true
		} else {
			for _, c := range collisions {
				if slices.Contains(c, e) {
					isColliding = true
					break
				}
			}
		}

		if isColliding {
			lineColor = data.Debug_ColliderCollidedColor
		}

		v0 := utils.Vec2{
			X: tra.Pos.X + col.Vertices[len(col.Vertices)-1].X,
			Y: tra.Pos.Y + col.Vertices[len(col.Vertices)-1].Y,
		}
		v1 := utils.Vec2{
			X: tra.Pos.X + col.Vertices[0].X,
			Y: tra.Pos.Y + col.Vertices[0].Y,
		}

		vector.StrokeLine(
			screen,
			float32(v0.X-camera.X),
			float32(v0.Y-camera.Y),
			float32(v1.X-camera.X),
			float32(v1.Y-camera.Y),
			1,
			lineColor,
			false,
		)
		for i := 0; i < len(col.Vertices)-1; i++ {
			v0 = utils.Vec2{
				X: tra.Pos.X + col.Vertices[i].X,
				Y: tra.Pos.Y + col.Vertices[i].Y,
			}
			v1 = utils.Vec2{
				X: tra.Pos.X + col.Vertices[i+1].X,
				Y: tra.Pos.Y + col.Vertices[i+1].Y,
			}

			vector.StrokeLine(
				screen,
				float32(v0.X-camera.X),
				float32(v0.Y-camera.Y),
				float32(v1.X-camera.X),
				float32(v1.Y-camera.Y),
				1,
				lineColor,
				false,
			)
		}
	}

	return nil
}

func DrawAABBs(
	screen *ebiten.Image,
	camera utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	aabbcollisions map[ecscommon.EntityId][]ecscommon.EntityId,
) error {
	for e, col := range colliders {
		tra, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Collider",
				MissingComponent: "Transform",
			}
		}

		lineColor := data.Debug_AABBColliderColor

		isColliding := false
		if _, ok := aabbcollisions[e]; ok {
			isColliding = true
		} else {
			for _, c := range aabbcollisions {
				if slices.Contains(c, e) {
					isColliding = true
					break
				}
			}
		}

		if isColliding {
			lineColor = data.Debug_AABBColliderCollidedColor
		}

		verts := []utils.Vec2{
			utils.Vec2{X: tra.Pos.X + col.AABB[0].X, Y: tra.Pos.Y + col.AABB[0].Y},
			utils.Vec2{X: tra.Pos.X + col.AABB[1].X, Y: tra.Pos.Y + col.AABB[0].Y},
			utils.Vec2{X: tra.Pos.X + col.AABB[1].X, Y: tra.Pos.Y + col.AABB[1].Y},
			utils.Vec2{X: tra.Pos.X + col.AABB[0].X, Y: tra.Pos.Y + col.AABB[1].Y},
			utils.Vec2{X: tra.Pos.X + col.AABB[0].X, Y: tra.Pos.Y + col.AABB[0].Y},
		}

		for i := 0; i < len(verts)-1; i++ {
			vector.StrokeLine(
				screen,
				float32(verts[i].X-camera.X),
				float32(verts[i].Y-camera.Y),
				float32(verts[i+1].X-camera.X),
				float32(verts[i+1].Y-camera.Y),
				1,
				lineColor,
				false,
			)
		}
	}

	return nil
}
