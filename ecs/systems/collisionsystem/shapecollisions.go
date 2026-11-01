package collisionsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"log"
	"math"
)

// Pass -1 in case of collision without entity
func GetRectangleCircleCollision(
	rEnt common.EntityId,
	cEnt common.EntityId,
	rHit shapes.RectangleShape,
	cHit shapes.CircleShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	var err error

	var cWorldPos, rWorldPos utils.Vec2f

	if cEnt != -1 {
		cWorldPos, err = ecsContainer.WorldPos(cEnt)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	if rEnt != -1 {
		rWorldPos, err = ecsContainer.WorldPos(rEnt)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", rEnt, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	circleStart := utils.Vec2f{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	rectMin := utils.Vec2f{X: rWorldPos.X + rHit.GetAABB()[0].X, Y: rWorldPos.Y + rHit.GetAABB()[0].Y}
	rectMax := utils.Vec2f{X: rWorldPos.X + rHit.GetAABB()[1].X, Y: rWorldPos.Y + rHit.GetAABB()[1].Y}

	if v, inside := circleCenterInRectangle(circleStart, cHit.GetRadius(), rectMin, rectMax); inside {
		return v
	}

	cVel := localVelocity(cEnt, ecsContainer)
	rVel := localVelocity(rEnt, ecsContainer)

	relVel := cVel.Subtract(rVel)

	if relVel.IsZero() {
		closestPoint := utils.Vec2f{
			X: utils.Clamp(circleStart.X, rectMin.X, rectMax.X),
			Y: utils.Clamp(circleStart.Y, rectMin.Y, rectMax.Y),
		}
		collisionVector := circleStart.Subtract(closestPoint)
		distance := collisionVector.Length()
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 && distance != 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
		return utils.Vec2f{X: 0, Y: 0}
	}

	// Expand rectangle by circle radius (Minkowski sum)
	expandedMin := rectMin.Subtract(utils.Vec2f{X: cHit.GetRadius(), Y: cHit.GetRadius()})
	expandedMax := rectMax.Add(utils.Vec2f{X: cHit.GetRadius(), Y: cHit.GetRadius()})

	// Ray-AABB intersection (slab method)
	tEntry := 0.0
	tExit := 1.0
	for _, axis := range []string{"X", "Y"} {
		var start, min, max, dir float64
		if axis == "X" {
			start = circleStart.X
			min = expandedMin.X
			max = expandedMax.X
			dir = relVel.X
		} else {
			start = circleStart.Y
			min = expandedMin.Y
			max = expandedMax.Y
			dir = relVel.Y
		}
		if dir == 0 {
			if start < min || start > max {
				return utils.Vec2f{X: 0, Y: 0}
			}
		} else {
			t1 := (min - start) / dir
			t2 := (max - start) / dir
			tMin := math.Min(t1, t2)
			tMax := math.Max(t1, t2)
			if tMin > tEntry {
				tEntry = tMin
			}
			if tMax < tExit {
				tExit = tMax
			}
			if tEntry > tExit || tExit < 0 || tEntry > 1 {
				return utils.Vec2f{X: 0, Y: 0}
			}
		}
	}

	// Collision occurs at tEntry
	if tEntry >= 0 && tEntry <= 1 {
		contactPoint := circleStart.Add(relVel.Multiply(tEntry))
		closestPoint := utils.Vec2f{
			X: utils.Clamp(contactPoint.X, rectMin.X, rectMax.X),
			Y: utils.Clamp(contactPoint.Y, rectMin.Y, rectMax.Y),
		}
		collisionVector := contactPoint.Subtract(closestPoint)
		distance := collisionVector.Length()
		if distance == 0 {
			// Touching edge, push out perpendicular to movement
			if math.Abs(relVel.X) > math.Abs(relVel.Y) {
				collisionVector = utils.Vec2f{X: relVel.X, Y: 0}
			} else {
				collisionVector = utils.Vec2f{X: 0, Y: relVel.Y}
			}
			if collisionVector.IsZero() {
				return utils.Vec2f{X: 0, Y: 0}
			}
			return collisionVector.Normalized().Multiply(0.01) // Small nudge
		}
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
	}
	return utils.Vec2f{X: 0, Y: 0}
}

// The closest point on the rectangle is the center itself then, so the distance can't tell which way is out; the
// circle leaves through the nearest edge instead
func circleCenterInRectangle(center utils.Vec2f, radius float64, rectMin, rectMax utils.Vec2f) (utils.Vec2f, bool) {
	if center.X <= rectMin.X || center.X >= rectMax.X || center.Y <= rectMin.Y || center.Y >= rectMax.Y {
		return utils.Vec2f{}, false
	}

	exits := []utils.Vec2f{
		{X: -(center.X - rectMin.X)},
		{X: rectMax.X - center.X},
		{Y: -(center.Y - rectMin.Y)},
		{Y: rectMax.Y - center.Y},
	}

	nearest := exits[0]
	for _, e := range exits[1:] {
		if e.Length() < nearest.Length() {
			nearest = e
		}
	}

	return nearest.Add(nearest.Normalized().Multiply(radius)), true
}

// Pass -1 in case of collision without entity
func GetRectangleRectangleCollision(
	r1Ent common.EntityId,
	r2Ent common.EntityId,
	r1Hit shapes.RectangleShape,
	r2Hit shapes.RectangleShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	var err error

	var r1WorldPos, r2WorldPos utils.Vec2f

	if r1Ent != -1 {
		r1WorldPos, err = ecsContainer.WorldPos(r1Ent)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", r1Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}
	if r2Ent != -1 {
		r2WorldPos, err = ecsContainer.WorldPos(r2Ent)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", r2Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	r1Min := utils.Vec2f{X: r1WorldPos.X + r1Hit.GetAABB()[0].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[0].Y}
	r1Max := utils.Vec2f{X: r1WorldPos.X + r1Hit.GetAABB()[1].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[1].Y}
	r2Min := utils.Vec2f{X: r2WorldPos.X + r2Hit.GetAABB()[0].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[0].Y}
	r2Max := utils.Vec2f{X: r2WorldPos.X + r2Hit.GetAABB()[1].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[1].Y}

	r1Vel := localVelocity(r1Ent, ecsContainer)
	r2Vel := localVelocity(r2Ent, ecsContainer)

	relVel := r1Vel.Subtract(r2Vel)

	if relVel.IsZero() {
		overlapX := math.Min(r1Max.X, r2Max.X) - math.Max(r1Min.X, r2Min.X)
		overlapY := math.Min(r1Max.Y, r2Max.Y) - math.Max(r1Min.Y, r2Min.Y)
		if overlapX > 0 && overlapY > 0 {
			if overlapX < overlapY {
				sign := 1.0
				if r2Min.X < r1Min.X {
					sign = -1.0
				}
				return utils.Vec2f{X: overlapX * sign, Y: 0}
			} else {
				sign := 1.0
				if r2Min.Y < r1Min.Y {
					sign = -1.0
				}
				return utils.Vec2f{X: 0, Y: overlapY * sign}
			}
		}
		return utils.Vec2f{X: 0, Y: 0}
	}

	// Swept AABB (slab method)
	tEntry := 0.0
	tExit := 1.0
	for _, axis := range []string{"X", "Y"} {
		var r1Start, r1End, r2Start, r2End, dir float64
		if axis == "X" {
			r1Start = r1Min.X
			r1End = r1Max.X
			r2Start = r2Min.X
			r2End = r2Max.X
			dir = relVel.X
		} else {
			r1Start = r1Min.Y
			r1End = r1Max.Y
			r2Start = r2Min.Y
			r2End = r2Max.Y
			dir = relVel.Y
		}
		if dir == 0 {
			if r1End < r2Start || r1Start > r2End {
				return utils.Vec2f{X: 0, Y: 0}
			}
		} else {
			t1 := (r2Start - r1End) / dir
			t2 := (r2End - r1Start) / dir
			tMin := math.Min(t1, t2)
			tMax := math.Max(t1, t2)
			if tMin > tEntry {
				tEntry = tMin
			}
			if tMax < tExit {
				tExit = tMax
			}
			if tEntry > tExit || tExit < 0 || tEntry > 1 {
				return utils.Vec2f{X: 0, Y: 0}
			}
		}
	}

	// Collision occurs at tEntry
	if tEntry >= 0 && tEntry <= 1 {
		// Move r1 to contact point
		contactMin := r1Min.Add(relVel.Multiply(tEntry))
		contactMax := r1Max.Add(relVel.Multiply(tEntry))
		overlapX := math.Min(contactMax.X, r2Max.X) - math.Max(contactMin.X, r2Min.X)
		overlapY := math.Min(contactMax.Y, r2Max.Y) - math.Max(contactMin.Y, r2Min.Y)
		if overlapX > 0 && overlapY > 0 {
			if overlapX < overlapY {
				sign := 1.0
				if relVel.X < 0 {
					sign = -1.0
				}
				return utils.Vec2f{X: overlapX * sign, Y: 0}
			} else {
				sign := 1.0
				if relVel.Y < 0 {
					sign = -1.0
				}
				return utils.Vec2f{X: 0, Y: overlapY * sign}
			}
		}
	}
	return utils.Vec2f{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetCircleCircleCollision(
	c1Ent common.EntityId,
	c2Ent common.EntityId,
	c1Hit shapes.CircleShape,
	c2Hit shapes.CircleShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	var err error

	var c1WorldPos, c2WorldPos utils.Vec2f

	if c1Ent != -1 {
		c1WorldPos, err = ecsContainer.WorldPos(c1Ent)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", c1Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	if c2Ent != -1 {
		c2WorldPos, err = ecsContainer.WorldPos(c2Ent)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", c2Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	center1 := utils.Vec2f{X: c1WorldPos.X + c1Hit.GetOffset().X, Y: c1WorldPos.Y + c1Hit.GetOffset().Y}
	center2 := utils.Vec2f{X: c2WorldPos.X + c2Hit.GetOffset().X, Y: c2WorldPos.Y + c2Hit.GetOffset().Y}

	c1Vel := localVelocity(c1Ent, ecsContainer)
	c2Vel := localVelocity(c2Ent, ecsContainer)

	relVel := c1Vel.Subtract(c2Vel)
	r := c1Hit.GetRadius() + c2Hit.GetRadius()

	// 1. Check for initial overlap (resolves static overlaps and handles relVel == 0)
	initialVector := center2.Subtract(center1)
	initialDist := initialVector.Length()
	if initialDist < r {
		penetrationDepth := r - initialDist
		if initialDist != 0 {
			return initialVector.Normalized().Multiply(penetrationDepth)
		}
		return utils.Vec2f{X: penetrationDepth, Y: 0}
	}

	if relVel.IsZero() {
		return utils.Vec2f{X: 0, Y: 0}
	}

	// 2. CCD: Sweep circle1 along relVel against circle2
	f := center1.Subtract(center2)

	a := relVel.Dot(relVel)
	b := 2 * f.Dot(relVel)
	c := f.Dot(f) - r*r

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return utils.Vec2f{X: 0, Y: 0}
	}
	sqrtDisc := math.Sqrt(discriminant)

	// Since we handled initial overlap (c < 0), t1 is guaranteed to be the entry point
	t1 := (-b - sqrtDisc) / (2 * a)

	if t1 < 0 || t1 > 1 {
		return utils.Vec2f{X: 0, Y: 0}
	}

	contact1 := center1.Add(relVel.Multiply(t1))
	collisionNormal := center2.Subtract(contact1)
	if collisionNormal.Length() == 0 {
		return utils.Vec2f{X: 0, Y: 0}
	}
	normal := collisionNormal.Normalized()

	// Penetration depth is the amount of movement remaining AFTER contact,
	// projected onto the contact normal. This handles pass-through correctly too.
	overshoot := relVel.Multiply(1 - t1)
	penetrationDepth := overshoot.Dot(normal)

	if penetrationDepth > 0 {
		return normal.Multiply(penetrationDepth)
	}

	return utils.Vec2f{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetRectanglePolygonCollision(
	rEnt common.EntityId,
	pEnt common.EntityId,
	rHit shapes.RectangleShape,
	pHit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	rectAsPolygon, err := shapes.NewPolygonShape(
		[]utils.Vec2f{
			utils.Vec2f{X: rHit.GetAABB()[0].X, Y: rHit.GetAABB()[0].Y},
			utils.Vec2f{X: rHit.GetAABB()[1].X, Y: rHit.GetAABB()[0].Y},
			utils.Vec2f{X: rHit.GetAABB()[1].X, Y: rHit.GetAABB()[1].Y},
			utils.Vec2f{X: rHit.GetAABB()[0].X, Y: rHit.GetAABB()[1].Y},
		},
		// The rectangle's AABB already includes its offset
		utils.Vec2f{},
	)

	if err != nil {
		log.Printf("error converting rectangle to polygon for collision detection: %v", err)
		return utils.Vec2f{X: 0, Y: 0}
	}

	return GetPolygonPolygonCollision(rEnt, pEnt, *rectAsPolygon, pHit, ecsContainer)
}

// Pass -1 in case of collision without entity
func GetCirclePolygonCollision(
	cEnt common.EntityId,
	pEnt common.EntityId,
	cHit shapes.CircleShape,
	pHit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	var err error

	var cWorldPos, pWorldPos utils.Vec2f

	if cEnt != -1 {
		cWorldPos, err = ecsContainer.WorldPos(cEnt)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	if pEnt != -1 {
		pWorldPos, err = ecsContainer.WorldPos(pEnt)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", pEnt, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	circleStart := utils.Vec2f{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	polyVerts := GetWorldPolygonVertices(pHit, pWorldPos)

	cVel := localVelocity(cEnt, ecsContainer)
	pVel := localVelocity(pEnt, ecsContainer)

	relVel := cVel.Subtract(pVel)

	if relVel.IsZero() {
		return circlePolygonPushOut(circleStart, cHit.GetRadius(), polyVerts)
	}

	// Swept test: move the circle along relVel and check for the first overlap, like polygon-polygon
	steps := 4
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		if v := circlePolygonPushOut(circleStart.Add(relVel.Multiply(t)), cHit.GetRadius(), polyVerts); !v.IsZero() {
			return v
		}
	}
	return utils.Vec2f{X: 0, Y: 0}
}

// Pushes the polygon out of the circle across the polygon edge closest to the circle's center
func circlePolygonPushOut(center utils.Vec2f, radius float64, verts []utils.Vec2f) utils.Vec2f {
	closest, closestEdge := utils.Vec2f{}, 0
	minDist := math.MaxFloat64
	for i, a := range verts {
		p := closestPointOnSegment(center, a, verts[(i+1)%len(verts)])
		if d := p.Subtract(center).Length(); d < minDist {
			minDist, closest, closestEdge = d, p, i
		}
	}

	inside := pointInConvexPolygon(center, verts)
	if !inside && minDist >= radius {
		return utils.Vec2f{X: 0, Y: 0}
	}

	var outward utils.Vec2f
	if minDist == 0 {
		outward = edgeOutwardNormal(verts, closestEdge)
	} else if inside {
		outward = closest.Subtract(center).Normalized()
	} else {
		outward = center.Subtract(closest).Normalized()
	}

	depth := radius - minDist
	if inside {
		depth = radius + minDist
	}

	return outward.Multiply(-depth)
}

func closestPointOnSegment(p, a, b utils.Vec2f) utils.Vec2f {
	ab := b.Subtract(a)
	lenSq := ab.Dot(ab)
	if lenSq == 0 {
		return a
	}
	t := utils.Clamp(p.Subtract(a).Dot(ab)/lenSq, 0, 1)
	return a.Add(ab.Multiply(t))
}

// Works for either winding, which is why it compares the signs of the edges rather than expecting one
func pointInConvexPolygon(p utils.Vec2f, verts []utils.Vec2f) bool {
	pos, neg := false, false
	for i, a := range verts {
		b := verts[(i+1)%len(verts)]
		cross := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
		pos = pos || cross > 0
		neg = neg || cross < 0
	}
	return !(pos && neg)
}

func edgeOutwardNormal(verts []utils.Vec2f, edge int) utils.Vec2f {
	a, b := verts[edge], verts[(edge+1)%len(verts)]
	normal := utils.Vec2f{X: -(b.Y - a.Y), Y: b.X - a.X}.Normalized()
	if normal.Dot(a.Subtract(polygonCentroid(verts))) < 0 {
		normal = normal.Multiply(-1)
	}
	return normal
}

// Pass -1 in case of collision without entity
func GetPolygonPolygonCollision(
	p1Ent common.EntityId,
	p2Ent common.EntityId,
	p1Hit shapes.PolygonShape,
	p2Hit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2f {
	var err error

	var p1WorldPos, p2WorldPos utils.Vec2f

	if p1Ent != -1 {
		p1WorldPos, err = ecsContainer.WorldPos(p1Ent)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", p1Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	if p2Ent != -1 {
		p2WorldPos, err = ecsContainer.WorldPos(p2Ent)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", p2Ent, err)
			return utils.Vec2f{X: 0, Y: 0}
		}
	}

	p1Verts := GetWorldPolygonVertices(p1Hit, p1WorldPos)
	p2Verts := GetWorldPolygonVertices(p2Hit, p2WorldPos)

	p1Vel := localVelocity(p1Ent, ecsContainer)
	p2Vel := localVelocity(p2Ent, ecsContainer)

	// Relative movement
	relVel := p1Vel.Subtract(p2Vel)
	if relVel.IsZero() {
		// Fallback to static SAT overlap check
		mtv, overlap := PolygonSAT(p1Verts, p2Verts)
		if overlap {
			return mtv
		}
		return utils.Vec2f{X: 0, Y: 0}
	}

	// Swept SAT: conservative approach, move p1 along relVel and check for first overlap
	steps := 4
	minT := 1.1
	var mtv utils.Vec2f
	found := false
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		offset := relVel.Multiply(t)
		movedVerts := make([]utils.Vec2f, len(p1Verts))
		for j, v := range p1Verts {
			movedVerts[j] = v.Add(offset)
		}
		mtvStep, overlap := PolygonSAT(movedVerts, p2Verts)
		if overlap {
			if t < minT {
				minT = t
				mtv = mtvStep
				found = true
			}
			break
		}
	}
	if found {
		return mtv
	}
	return utils.Vec2f{X: 0, Y: 0}
}

func GetWorldPolygonVertices(p shapes.PolygonShape, worldPos utils.Vec2f) []utils.Vec2f {
	verts := p.GetVertices()
	origin := worldPos.Add(p.GetOffset())
	worldVerts := make([]utils.Vec2f, len(verts))
	for i, v := range verts {
		worldVerts[i] = origin.Add(v)
	}
	return worldVerts
}

// PolygonSAT returns the minimum translation vector (MTV) and whether two convex polygons overlap.
// vertsA and vertsB must be in world space and ordered (clockwise or counterclockwise).
func PolygonSAT(vertsA, vertsB []utils.Vec2f) (utils.Vec2f, bool) {
	var mtv utils.Vec2f
	minOverlap := math.MaxFloat64
	var smallestAxis utils.Vec2f

	axes := getPolygonAxes(vertsA)
	axes = append(axes, getPolygonAxes(vertsB)...)

	for _, axis := range axes {
		minA, maxA := projectPolygon(axis, vertsA)
		minB, maxB := projectPolygon(axis, vertsB)

		overlap := math.Min(maxA, maxB) - math.Max(minA, minB)
		if overlap <= 0 {
			return utils.Vec2f{X: 0, Y: 0}, false
		}
		if overlap < minOverlap {
			minOverlap = overlap
			smallestAxis = axis
		}
	}

	// MTV points from A to B
	centerA := polygonCentroid(vertsA)
	centerB := polygonCentroid(vertsB)
	dir := centerB.Subtract(centerA)
	if dir.Dot(smallestAxis) < 0 {
		smallestAxis = smallestAxis.Multiply(-1)
	}
	mtv = smallestAxis.Normalized().Multiply(minOverlap)
	return mtv, true
}

func getPolygonAxes(verts []utils.Vec2f) []utils.Vec2f {
	axes := make([]utils.Vec2f, len(verts))
	for i := 0; i < len(verts); i++ {
		j := (i + 1) % len(verts)
		edge := verts[j].Subtract(verts[i])
		normal := utils.Vec2f{X: -edge.Y, Y: edge.X}.Normalized()
		axes[i] = normal
	}
	return axes
}

func projectPolygon(axis utils.Vec2f, verts []utils.Vec2f) (float64, float64) {
	min := verts[0].Dot(axis)
	max := min
	for _, v := range verts[1:] {
		p := v.Dot(axis)
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}
	return min, max
}

func polygonCentroid(verts []utils.Vec2f) utils.Vec2f {
	var c utils.Vec2f
	for _, v := range verts {
		c = c.Add(v)
	}
	return c.Multiply(1.0 / float64(len(verts)))
}

// Entities without a velocity, and shapes without an entity, count as not moving
func localVelocity(e common.EntityId, ecsContainer *ecs.ECSContainer) utils.Vec2f {
	if v, ok := ecsContainer.Velocities.Get(e); ok {
		return v.LocalVector()
	}
	return utils.Vec2f{}
}
