package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawCollisions(
	screen *ebiten.Image,
	camera utils.Vec2,
	collisions map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	for eA, cols := range collisions {
		tm := components.TransformManager{}

		aWorldPos, err := tm.GetWorldPos(eA,transforms, parents)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", eA, err)
			continue
		}

		for _, colVector := range cols {
			vector.StrokeLine(
				screen,
				float32(aWorldPos.X-camera.X),
				float32(aWorldPos.Y-camera.Y),
				float32(aWorldPos.X+colVector.X*10-camera.X),
				float32(aWorldPos.Y+colVector.Y*10-camera.Y),
				2,
				data.Debug_CollisionVectorColor,
				false,
			)
		}
	}

	return nil
}

func DrawColliders(
	screen *ebiten.Image,
	camera utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	collisions map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	for e, col := range colliders {
		tm := components.TransformManager{}

		worldPos, err := tm.GetWorldPos(e, transforms, parents)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
		}

		lineColor := data.Debug_ColliderColor

		if _, ok := collisions[e]; ok {
			lineColor = data.Debug_ColliderCollidedColor
		} else {
			for _, c := range collisions {
				if _, ok := c[e]; ok {
					lineColor = data.Debug_ColliderCollidedColor
					break
				}
			}
		}

		for _, hitbox := range col.Hitboxes {
			switch h := hitbox.(type) {
			case *hitboxes.RectangleHitbox:
				verts := []utils.Vec2{
					utils.Vec2{X: worldPos.X + h.GetOffset().X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetOffset().Y + h.GetAABB()[0].Y},
					utils.Vec2{X: worldPos.X + h.GetOffset().X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetOffset().Y + h.GetAABB()[0].Y},
					utils.Vec2{X: worldPos.X + h.GetOffset().X + h.GetAABB()[1].X, Y: worldPos.Y + h.GetOffset().Y + h.GetAABB()[1].Y},
					utils.Vec2{X: worldPos.X + h.GetOffset().X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetOffset().Y + h.GetAABB()[1].Y},
					utils.Vec2{X: worldPos.X + h.GetOffset().X + h.GetAABB()[0].X, Y: worldPos.Y + h.GetOffset().Y + h.GetAABB()[0].Y},
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
			case *hitboxes.CircleHitbox:
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
			case *hitboxes.PolygonHitbox:
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
	screen *ebiten.Image,
	camera utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
	aabbcollisions map[ecscommon.EntityId][]ecscommon.EntityId,
) error {
	for e, col := range colliders {
		tm := components.TransformManager{}

		worldPos, err := tm.GetWorldPos(e, transforms, parents)
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			continue
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
			utils.Vec2{X: worldPos.X + col.GetAABB()[0].X, Y: worldPos.Y + col.GetAABB()[0].Y},
			utils.Vec2{X: worldPos.X + col.GetAABB()[1].X, Y: worldPos.Y + col.GetAABB()[0].Y},
			utils.Vec2{X: worldPos.X + col.GetAABB()[1].X, Y: worldPos.Y + col.GetAABB()[1].Y},
			utils.Vec2{X: worldPos.X + col.GetAABB()[0].X, Y: worldPos.Y + col.GetAABB()[1].Y},
			utils.Vec2{X: worldPos.X + col.GetAABB()[0].X, Y: worldPos.Y + col.GetAABB()[0].Y},
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
