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
	potentialCollisions map[ecscommon.Entity][]ecscommon.Entity,
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
) (map[ecscommon.Entity][]ecscommon.Entity, error) {
	collisions := make(map[ecscommon.Entity][]ecscommon.Entity)

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
				return nil, &ecscommon.ErrorMissingComponent{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponent{
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
					collisions[eA] = []ecscommon.Entity{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	proximateEntities map[ecscommon.Entity][]ecscommon.Entity,
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
) (map[ecscommon.Entity][]ecscommon.Entity, error) {
	collisions := make(map[ecscommon.Entity][]ecscommon.Entity)

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
				return nil, &ecscommon.ErrorMissingComponent{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponent{
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
					collisions[eA] = []ecscommon.Entity{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}

func GetSHGProximities(
	grid map[ecscommon.CellKey][]ecscommon.Entity,
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
) (map[ecscommon.Entity][]ecscommon.Entity, error) {
	proximateEntities := make(map[ecscommon.Entity][]ecscommon.Entity)

	for eA, _ := range colliders {
		traA, ok := transforms[eA]
		if !ok {
			return nil, &ecscommon.ErrorMissingComponent{
				Entity:           eA,
				PresentComponent: "Collider",
				MissingComponent: "Transform",
			}
		}
		cellX := int(traA.Pos.X / data.SpatialHashGridCellSize)
		cellY := int(traA.Pos.Y / data.SpatialHashGridCellSize)
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, eB := range grid[ecscommon.CellKey{X: cellX + dx, Y: cellY + dy}] {
					if eA == eB {
						continue
					}

					_, ok := colliders[eB]
					if !ok {
						continue
					}

					_, ok = transforms[eB]
					if !ok {
						return nil, &ecscommon.ErrorMissingComponent{
							Entity:           eB,
							PresentComponent: "Collider",
							MissingComponent: "Transform",
						}
					}

					if proximateEntity, ok := proximateEntities[eB]; ok {
						if slices.Contains(proximateEntity, eA) {
							continue
						}
					}

					if !slices.Contains(proximateEntities[eA], eB) {
						proximateEntities[eA] = append(proximateEntities[eA], eB)
					}
				}
			}
		}
	}

	return proximateEntities, nil
}

func PopulateSpatialHashGrid(
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
) (map[ecscommon.CellKey][]ecscommon.Entity, error) {
	grid := make(map[ecscommon.CellKey][]ecscommon.Entity)

	for e, col := range colliders {
		tra, ok := transforms[e]
		if !ok {
			return nil, &ecscommon.ErrorMissingComponent{
				Entity:           e,
				PresentComponent: "Collider",
				MissingComponent: "Transform",
			}
		}

		minCellX := int((tra.Pos.X + col.AABB[0].X) / data.SpatialHashGridCellSize)
		minCellY := int((tra.Pos.Y + col.AABB[0].Y) / data.SpatialHashGridCellSize)
		maxCellX := int((tra.Pos.X + col.AABB[1].X) / data.SpatialHashGridCellSize)
		maxCellY := int((tra.Pos.Y + col.AABB[1].Y) / data.SpatialHashGridCellSize)
		for x := minCellX; x <= maxCellX; x++ {
			for y := minCellY; y <= maxCellY; y++ {
				grid[ecscommon.CellKey{X: x, Y: y}] = append(grid[ecscommon.CellKey{X: x, Y: y}], e)
			}
		}
	}

	return grid, nil
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
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
	collisions map[ecscommon.Entity][]ecscommon.Entity,
) error {
	for e, col := range colliders {
		tra, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponent{
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
			float32(v0.X),
			float32(v0.Y),
			float32(v1.X),
			float32(v1.Y),
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
				float32(v0.X),
				float32(v0.Y),
				float32(v1.X),
				float32(v1.Y),
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
	colliders map[ecscommon.Entity]*components.Collider,
	transforms map[ecscommon.Entity]*components.Transform,
	aabbcollisions map[ecscommon.Entity][]ecscommon.Entity,
) error {
	for e, col := range colliders {
		tra, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponent{
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
				float32(verts[i].X),
				float32(verts[i].Y),
				float32(verts[i+1].X),
				float32(verts[i+1].Y),
				1,
				lineColor,
				false,
			)
		}
	}

	return nil
}
