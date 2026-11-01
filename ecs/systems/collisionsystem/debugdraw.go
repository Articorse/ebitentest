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
	camera utils.Vec2f,
	collisions []common.Collision,
	ecsContainer *ecs.ECSContainer,
) error {
	for _, col := range collisions {
		aWorldPos, err := ecsContainer.WorldPos(col.A)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", col.A, err)
			continue
		}

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

	return nil
}

func DrawColliders(
	colliders ecs.ColliderSet,
	baseColor color.RGBA,
	collidedColor color.RGBA,
	screen *ebiten.Image,
	camera utils.Vec2f,
	collisions []common.Collision,
	ecsContainer *ecs.ECSContainer,
) error {
	for _, e := range colliders.GetEntities() {
		col, _ := colliders.Collider(e)

		worldPos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		drawWindow := [2]utils.Vec2f{
			utils.Vec2f{X: 0 - data.SpatialHashGridCellSize, Y: 0 - data.SpatialHashGridCellSize},
			utils.Vec2f{X: data.CameraWidth + data.SpatialHashGridCellSize, Y: data.CameraHeight + data.SpatialHashGridCellSize},
		}

		if worldPos.X < drawWindow[0].X ||
			worldPos.X > drawWindow[1].X ||
			worldPos.Y < drawWindow[0].Y ||
			worldPos.Y > drawWindow[1].Y {
			continue
		}

		lineColor := baseColor

		if slices.ContainsFunc(collisions, func(c common.Collision) bool { return c.A == e || c.B == e }) {
			lineColor = collidedColor
		}

		for _, shape := range col.Shapes() {
			switch h := shape.(type) {
			case *shapes.RectangleShape:
				verts := []utils.Vec2f{
					utils.Vec2f{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[0].Y},
					utils.Vec2f{X: worldPos.X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetAABB()[0].Y},
					utils.Vec2f{X: worldPos.X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetAABB()[1].Y},
					utils.Vec2f{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[1].Y},
					utils.Vec2f{X: worldPos.X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetAABB()[0].Y},
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
				center := utils.Vec2f{X: worldPos.X + h.GetOffset().X, Y: worldPos.Y + h.GetOffset().Y}
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
				verts := GetWorldPolygonVertices(*h, worldPos)
				for i, v := range verts {
					next := verts[(i+1)%len(verts)]
					vector.StrokeLine(
						screen,
						float32(v.X-camera.X),
						float32(v.Y-camera.Y),
						float32(next.X-camera.X),
						float32(next.Y-camera.Y),
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
	colliders ecs.ColliderSet,
	baseColor color.RGBA,
	collidedColor color.RGBA,
	screen *ebiten.Image,
	camera utils.Vec2f,
	aabbcollisions []common.EntityPair,
	ecsContainer *ecs.ECSContainer,
) error {
	for _, e := range colliders.GetEntities() {
		col, _ := colliders.Collider(e)

		worldPos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		drawWindow := [2]utils.Vec2f{
			utils.Vec2f{X: 0 - data.SpatialHashGridCellSize, Y: 0 - data.SpatialHashGridCellSize},
			utils.Vec2f{X: data.CameraWidth + data.SpatialHashGridCellSize, Y: data.CameraHeight + data.SpatialHashGridCellSize},
		}

		if worldPos.X < drawWindow[0].X ||
			worldPos.X > drawWindow[1].X ||
			worldPos.Y < drawWindow[0].Y ||
			worldPos.Y > drawWindow[1].Y {
			continue
		}

		lineColor := baseColor

		if slices.ContainsFunc(aabbcollisions, func(p common.EntityPair) bool { return p.A == e || p.B == e }) {
			lineColor = collidedColor
		}

		aabb, err := ecsContainer.WorldAABB(e, col.LocalPaddedAABB())
		if err != nil {
			log.Printf("Error getting AABB for entity %d: %v\n", e, err)
			continue
		}

		verts := []utils.Vec2f{
			utils.Vec2f{X: aabb[0].X, Y: aabb[0].Y},
			utils.Vec2f{X: aabb[1].X, Y: aabb[0].Y},
			utils.Vec2f{X: aabb[1].X, Y: aabb[1].Y},
			utils.Vec2f{X: aabb[0].X, Y: aabb[1].Y},
			utils.Vec2f{X: aabb[0].X, Y: aabb[0].Y},
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
