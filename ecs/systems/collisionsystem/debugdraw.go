package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Outlines the colliders on screen, in the touching color for entities that touched something this tick
func DrawColliders(
	colliders ecs.ColliderSet,
	baseColor color.RGBA,
	touchingColor color.RGBA,
	screen *ebiten.Image,
	camera utils.Vec2f,
	touching map[common.EntityId]bool,
	ecsContainer *ecs.ECSContainer,
) {
	onScreen := [2]utils.Vec2f{camera, camera.Add(utils.Vec2f{X: data.CameraWidth, Y: data.CameraHeight})}

	for _, e := range colliders.GetEntities() {
		col, _ := colliders.Collider(e)
		if ecsContainer.IsDisabled(e) || !col.Enabled() {
			continue
		}

		aabb, err := ecsContainer.ColliderWorldAABB(e, col)
		if err != nil {
			log.Printf("Error getting AABB of entity %d: %v\n", e, err)
			continue
		}
		if !utils.DetectAABBCollision(aabb, onScreen) {
			continue
		}

		worldPos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		lineColor := baseColor
		if touching[e] {
			lineColor = touchingColor
		}

		for _, shape := range col.Shapes() {
			shape, ok := entityShape(e, shape, ecsContainer)
			if !ok {
				continue
			}
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
}
