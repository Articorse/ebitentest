package collisionsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/utils"
	"log"
	"math"
)

func getRectangleCircleCollision(
	r hitboxes.RectangleHitbox,
	c hitboxes.CircleHitbox,
	rCol components.Collider,
	cCol components.Collider,
	rTra components.Transform,
	cTra components.Transform,
) utils.Vec2 {
	closestPoint := utils.Vec2{
		X: utils.Clamp(cTra.GetPos().X, rTra.GetPos().X+r.GetAABB()[0].X, rTra.GetPos().X+r.GetAABB()[1].X),
		Y: utils.Clamp(cTra.GetPos().Y, rTra.GetPos().Y+r.GetAABB()[0].Y, rTra.GetPos().Y+r.GetAABB()[1].Y),
	}

	collisionVector := cTra.GetPos().Subtract(closestPoint)
	distance := collisionVector.Length()

	if distance == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}

	penetrationDepth := c.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getRectangleRectangleCollision(
	r1 hitboxes.RectangleHitbox,
	r2 hitboxes.RectangleHitbox,
	r1Col components.Collider,
	r2Col components.Collider,
	r1Tra components.Transform,
	r2Tra components.Transform,
) utils.Vec2 {
	rect1Center := utils.Vec2{X: r1Tra.GetPos().X + r1.GetOffset().X, Y: r1Tra.GetPos().Y + r1.GetOffset().Y}
	rect2Center := utils.Vec2{X: r2Tra.GetPos().X + r2.GetOffset().X, Y: r2Tra.GetPos().Y + r2.GetOffset().Y}

	delta := rect2Center.Subtract(rect1Center)
	overlapX := (r1.GetAABB()[1].X-r1.GetAABB()[0].X)/2 + (r2.GetAABB()[1].X-r2.GetAABB()[0].X)/2 - math.Abs(delta.X)
	overlapY := (r1.GetAABB()[1].Y-r1.GetAABB()[0].Y)/2 + (r2.GetAABB()[1].Y-r2.GetAABB()[0].Y)/2 - math.Abs(delta.Y)

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
	c1 hitboxes.CircleHitbox,
	c2 hitboxes.CircleHitbox,
	c1Col components.Collider,
	c2Col components.Collider,
	c1Tra components.Transform,
	c2Tra components.Transform,
) utils.Vec2 {
	center1 := utils.Vec2{X: c1Tra.GetPos().X + c1.GetOffset().X, Y: c1Tra.GetPos().Y + c1.GetOffset().Y}
	center2 := utils.Vec2{X: c2Tra.GetPos().X + c2.GetOffset().X, Y: c2Tra.GetPos().Y + c2.GetOffset().Y}

	collisionVector := center2.Subtract(center1)
	distance := collisionVector.Length()
	penetrationDepth := c1.GetRadius() + c2.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getRectanglePolygonCollision(
	r hitboxes.RectangleHitbox,
	p hitboxes.PolygonHitbox,
	rCol components.Collider,
	pCol components.Collider,
	rTra components.Transform,
	pTra components.Transform,
) utils.Vec2 {
	rectAsPolygon, err := hitboxes.NewPolygonHitbox(
		[]utils.Vec2{
			utils.Vec2{X: r.GetAABB()[0].X, Y: r.GetAABB()[0].Y},
			utils.Vec2{X: r.GetAABB()[1].X, Y: r.GetAABB()[0].Y},
			utils.Vec2{X: r.GetAABB()[1].X, Y: r.GetAABB()[1].Y},
			utils.Vec2{X: r.GetAABB()[0].X, Y: r.GetAABB()[1].Y},
		},
		r.GetOffset(),
	)

	if err != nil {
		log.Printf("error converting rectangle to polygon for collision detection: %v", err)
		return utils.Vec2{X: 0, Y: 0}
	}

	return getPolygonPolygonCollision(*rectAsPolygon, p, rCol, pCol, rTra, pTra)
}

func getCirclePolygonCollision(
	c hitboxes.CircleHitbox,
	p hitboxes.PolygonHitbox,
	cCol components.Collider,
	pCol components.Collider,
	cTra components.Transform,
	pTra components.Transform,
) utils.Vec2 {
	circleCenter := utils.Vec2{X: cTra.GetPos().X + c.GetOffset().X, Y: cTra.GetPos().Y + c.GetOffset().Y}

	var closestPoint utils.Vec2
	minDistance := math.MaxFloat64

	for _, v := range p.GetVertices() {
		vertexGlobal := utils.Vec2{X: pTra.GetPos().X + v.X, Y: pTra.GetPos().Y + v.Y}
		distance := vertexGlobal.Subtract(circleCenter).Length()
		if distance < minDistance {
			minDistance = distance
			closestPoint = vertexGlobal
		}
	}

	collisionVector := circleCenter.Subtract(closestPoint)
	distance := collisionVector.Length()
	penetrationDepth := c.GetRadius() - distance

	if penetrationDepth > 0 {
		return collisionVector.Normalized().Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

func getPolygonPolygonCollision(
	p1 hitboxes.PolygonHitbox,
	p2 hitboxes.PolygonHitbox,
	p1Col components.Collider,
	p2Col components.Collider,
	p1Tra components.Transform,
	p2Tra components.Transform,
) utils.Vec2 {
	closestPointP1 := utils.Vec2{}
	closestPointP2 := utils.Vec2{}
	minDistance := math.MaxFloat64

	for _, v1 := range p1.GetVertices() {
		vertexGlobal1 := utils.Vec2{X: p1Tra.GetPos().X + v1.X, Y: p1Tra.GetPos().Y + v1.Y}
		for _, v2 := range p2.GetVertices() {
			vertexGlobal2 := utils.Vec2{X: p2Tra.GetPos().X + v2.X, Y: p2Tra.GetPos().Y + v2.Y}
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
