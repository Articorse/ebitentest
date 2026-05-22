package collisionsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
	"math"
)

func getRectangleCircleCollision(
	rEnt ecscommon.EntityId,
	cEnt ecscommon.EntityId,
	rHit hitboxes.RectangleHitbox,
	cHit hitboxes.CircleHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) utils.Vec2 {
	tm := components.TransformManager{}

	cWorldPos, err := tm.GetWorldPos(cEnt, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	rWorldPos, err := tm.GetWorldPos(rEnt, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", rEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	closestPoint := utils.Vec2{
		X: utils.Clamp(cWorldPos.X, rWorldPos.X+rHit.GetAABB()[0].X, rWorldPos.X+rHit.GetAABB()[1].X),
		Y: utils.Clamp(cWorldPos.Y, rWorldPos.Y+rHit.GetAABB()[0].Y, rWorldPos.Y+rHit.GetAABB()[1].Y),
	}

	collisionVector := cWorldPos.Subtract(closestPoint)
	distance := collisionVector.Length()

	if distance == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}

	penetrationDepth := cHit.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getRectangleRectangleCollision(
	r1Ent ecscommon.EntityId,
	r2Ent ecscommon.EntityId,
	r1Hit hitboxes.RectangleHitbox,
	r2Hit hitboxes.RectangleHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) utils.Vec2 {
	tm := components.TransformManager{}

	r1WorldPos, err := tm.GetWorldPos(r1Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", r1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	r2WorldPos, err := tm.GetWorldPos(r2Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", r2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	rect1Center := utils.Vec2{X: r1WorldPos.X + r1Hit.GetOffset().X, Y: r1WorldPos.Y + r1Hit.GetOffset().Y}
	rect2Center := utils.Vec2{X: r2WorldPos.X + r2Hit.GetOffset().X, Y: r2WorldPos.Y + r2Hit.GetOffset().Y}

	delta := rect2Center.Subtract(rect1Center)
	overlapX := (r1Hit.GetAABB()[1].X-r1Hit.GetAABB()[0].X)/2 + (r2Hit.GetAABB()[1].X-r2Hit.GetAABB()[0].X)/2 - math.Abs(delta.X)
	overlapY := (r1Hit.GetAABB()[1].Y-r1Hit.GetAABB()[0].Y)/2 + (r2Hit.GetAABB()[1].Y-r2Hit.GetAABB()[0].Y)/2 - math.Abs(delta.Y)

	if overlapX > 0 && overlapY > 0 {
		if overlapX < overlapY {
			return utils.Vec2{X: overlapX * utils.Sign(delta.X), Y: 0}
		} else {
			return utils.Vec2{X: 0, Y: overlapY * utils.Sign(delta.Y)}
		}
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getCircleCircleCollision(
	c1Ent ecscommon.EntityId,
	c2Ent ecscommon.EntityId,
	c1Hit hitboxes.CircleHitbox,
	c2Hit hitboxes.CircleHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) utils.Vec2 {
	tm := components.TransformManager{}

	c1WorldPos, err := tm.GetWorldPos(c1Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", c1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	c2WorldPos, err := tm.GetWorldPos(c2Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", c2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	center1 := utils.Vec2{X: c1WorldPos.X + c1Hit.GetOffset().X, Y: c1WorldPos.Y + c1Hit.GetOffset().Y}
	center2 := utils.Vec2{X: c2WorldPos.X + c2Hit.GetOffset().X, Y: c2WorldPos.Y + c2Hit.GetOffset().Y}

	collisionVector := center2.Subtract(center1)
	distance := collisionVector.Length()
	penetrationDepth := c1Hit.GetRadius() + c2Hit.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getRectanglePolygonCollision(
	rEnt ecscommon.EntityId,
	pEnt ecscommon.EntityId,
	rHit hitboxes.RectangleHitbox,
	pHit hitboxes.PolygonHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) utils.Vec2 {
	rectAsPolygon, err := hitboxes.NewPolygonHitbox(
		[]utils.Vec2{
			utils.Vec2{X: rHit.GetAABB()[0].X, Y: rHit.GetAABB()[0].Y},
			utils.Vec2{X: rHit.GetAABB()[1].X, Y: rHit.GetAABB()[0].Y},
			utils.Vec2{X: rHit.GetAABB()[1].X, Y: rHit.GetAABB()[1].Y},
			utils.Vec2{X: rHit.GetAABB()[0].X, Y: rHit.GetAABB()[1].Y},
		},
		rHit.GetOffset(),
	)

	if err != nil {
		log.Printf("error converting rectangle to polygon for collision detection: %v", err)
		return utils.Vec2{X: 0, Y: 0}
	}

	return getPolygonPolygonCollision(rEnt, pEnt, *rectAsPolygon, pHit, transforms, parents)
}

func getCirclePolygonCollision(
	cEnt ecscommon.EntityId,
	pEnt ecscommon.EntityId,
	cHit hitboxes.CircleHitbox,
	pHit hitboxes.PolygonHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,	
) utils.Vec2 {
	tm := components.TransformManager{}

	cWorldPos, err := tm.GetWorldPos(cEnt, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	pWorldPos, err := tm.GetWorldPos(pEnt, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", pEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	circleCenter := utils.Vec2{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}

	var closestPoint utils.Vec2
	minDistance := math.MaxFloat64

	for _, v := range pHit.GetVertices() {
		vertexGlobal := utils.Vec2{X: pWorldPos.X + v.X, Y: pWorldPos.Y + v.Y}
		distance := vertexGlobal.Subtract(circleCenter).Length()
		if distance < minDistance {
			minDistance = distance
			closestPoint = vertexGlobal
		}
	}

	collisionVector := circleCenter.Subtract(closestPoint)
	distance := collisionVector.Length()
	penetrationDepth := cHit.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getPolygonPolygonCollision(
	p1Ent ecscommon.EntityId,
	p2Ent ecscommon.EntityId,
	p1Hit hitboxes.PolygonHitbox,
	p2Hit hitboxes.PolygonHitbox,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) utils.Vec2 {
	tm := components.TransformManager{}

	p1WorldPos, err := tm.GetWorldPos(p1Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", p1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	p2WorldPos, err := tm.GetWorldPos(p2Ent, transforms, parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", p2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	closestPointP1 := utils.Vec2{}
	closestPointP2 := utils.Vec2{}
	minDistance := math.MaxFloat64

	for _, v1 := range p1Hit.GetVertices() {
		vertexGlobal1 := utils.Vec2{X: p1WorldPos.X + v1.X, Y: p1WorldPos.Y + v1.Y}
		for _, v2 := range p2Hit.GetVertices() {
			vertexGlobal2 := utils.Vec2{X: p2WorldPos.X + v2.X, Y: p2WorldPos.Y + v2.Y}
			distance := vertexGlobal1.Subtract(vertexGlobal2).Length()
			if distance < minDistance {
				minDistance = distance
				closestPointP1 = vertexGlobal1
				closestPointP2 = vertexGlobal2
			}
		}
	}

	collisionVector := closestPointP1.Subtract(closestPointP2)
	distance := collisionVector.Length()

	if distance == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}

	penetrationDepth := minDistance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}
