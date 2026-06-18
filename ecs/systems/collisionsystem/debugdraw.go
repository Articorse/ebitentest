package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"image/color"
	"log"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawCollisions(
	screen *ebiten.Image,
	color color.RGBA,
	camera utils.Vec2,
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	world *ecs.World,
) error {
	for eA, cols := range collisions {
		tm := world.TransformManager

		if !world.Transforms.HasComponent(eA) {
			log.Printf("Entity %d in collisions does not have a transform component\n", eA)
			continue
		}

		aWorldPos, err := tm.GetWorldPos(eA, world)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", eA, err)
			continue
		}

		for _, col := range cols {
			vector.StrokeLine(
				screen,
				float32(aWorldPos.X-camera.X),
				float32(aWorldPos.Y-camera.Y),
				float32(aWorldPos.X+col.Vector.X*10-camera.X),
				float32(aWorldPos.Y+col.Vector.Y*10-camera.Y),
				2,
				color,
				false,
			)
		}
	}

	return nil
}

func DrawColliders(
	colManager ecs.IColliderManager,
	baseColor color.RGBA,
	collidedColor color.RGBA,
	screen *ebiten.Image,
	camera utils.Vec2,
	collisions map[common.EntityId]map[common.EntityId]common.Collision,
	world *ecs.World,
) error {
	for _, e := range colManager.EntityIds(world) {
		tm := world.TransformManager

		worldPos, err := tm.GetWorldPos(e, world)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		drawWindow := [2]utils.Vec2{
			utils.Vec2{X: 0 - data.SpatialHashGridCellSize, Y: 0 - data.SpatialHashGridCellSize},
			utils.Vec2{X: data.CameraWidth + data.SpatialHashGridCellSize, Y: data.CameraHeight + data.SpatialHashGridCellSize},
		}

		if worldPos.X < drawWindow[0].X ||
			worldPos.X > drawWindow[1].X ||
			worldPos.Y < drawWindow[0].Y ||
			worldPos.Y > drawWindow[1].Y {
			continue
		}

		lineColor := baseColor

		if _, ok := collisions[e]; ok {
			lineColor = collidedColor
		} else {
			for _, c := range collisions {
				if _, ok := c[e]; ok {
					lineColor = collidedColor
					break
				}
			}
		}

		colShapes, err := colManager.GetShapes(e, world)
		if err != nil {
			log.Printf("Error getting collider shapes for entity %d: %v\n", e, err)
			continue
		}

		for _, shape := range colShapes {
			switch h := shape.(type) {
			case *shapes.RectangleShape:
				verts := []utils.Vec2{
					utils.Vec2{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[0].Y},
					utils.Vec2{X: worldPos.X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetAABB()[0].Y},
					utils.Vec2{X: worldPos.X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetAABB()[1].Y},
					utils.Vec2{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[1].Y},
					utils.Vec2{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[0].Y},
				}
				for i := range verts[:len(verts)-1] {
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
			case *shapes.CircleShape:
				center := utils.Vec2{X: worldPos.X + h.GetOffset().X, Y: worldPos.Y + h.GetOffset().Y}
				vector.StrokeCircle(
					screen,
					float32(center.X-camera.X),
					float32(center.Y-camera.Y),
					float32(h.GetRadius()),
					1,
					lineColor,
					false,
				)
			case *shapes.PolygonShape:
				var verts []utils.Vec2
				for _, v := range h.GetVertices() {
					verts = append(verts, utils.Vec2{X: worldPos.X + v.X, Y: worldPos.Y + v.Y})
				}
				for _, v := range verts[:len(verts)-1] {
					vector.StrokeLine(
						screen,
						float32(v.X-camera.X),
						float32(v.Y-camera.Y),
						float32(v.X-camera.X),
						float32(v.Y-camera.Y),
						1,
						lineColor,
						false,
					)
				}
			}
		}
	}
	return nil
}

func DrawAABBs(
	colManager ecs.IColliderManager,
	baseColor color.RGBA,
	collidedColor color.RGBA,
	screen *ebiten.Image,
	camera utils.Vec2,
	aabbcollisions map[common.EntityId][]common.EntityId,
	world *ecs.World,
) error {
	for _, e := range colManager.EntityIds(world) {
		tm := world.TransformManager

		worldPos, err := tm.GetWorldPos(e, world)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		drawWindow := [2]utils.Vec2{
			utils.Vec2{X: 0 - data.SpatialHashGridCellSize, Y: 0 - data.SpatialHashGridCellSize},
			utils.Vec2{X: data.CameraWidth + data.SpatialHashGridCellSize, Y: data.CameraHeight + data.SpatialHashGridCellSize},
		}

		if worldPos.X < drawWindow[0].X ||
			worldPos.X > drawWindow[1].X ||
			worldPos.Y < drawWindow[0].Y ||
			worldPos.Y > drawWindow[1].Y {
			continue
		}

		lineColor := baseColor

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
			lineColor = collidedColor
		}

		aabb, err := colManager.GetWorldPaddedAABB(e, world)
		if err != nil {
			log.Printf("Error getting AABB for entity %d: %v\n", e, err)
			continue
		}

		verts := []utils.Vec2{
			utils.Vec2{X: aabb[0].X, Y: aabb[0].Y},
			utils.Vec2{X: aabb[1].X, Y: aabb[0].Y},
			utils.Vec2{X: aabb[1].X, Y: aabb[1].Y},
			utils.Vec2{X: aabb[0].X, Y: aabb[1].Y},
			utils.Vec2{X: aabb[0].X, Y: aabb[0].Y},
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
