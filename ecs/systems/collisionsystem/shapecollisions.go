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
) utils.Vec2 {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager
	var err error

	var cWorldPos, rWorldPos utils.Vec2

	if cEnt != -1 {
		cWorldPos, err = tm.GetWorldPos(cEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	if rEnt != -1 {
		rWorldPos, err = tm.GetWorldPos(rEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", rEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	circleStart := utils.Vec2{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	rectMin := utils.Vec2{X: rWorldPos.X + rHit.GetAABB()[0].X, Y: rWorldPos.Y + rHit.GetAABB()[0].Y}
	rectMax := utils.Vec2{X: rWorldPos.X + rHit.GetAABB()[1].X, Y: rWorldPos.Y + rHit.GetAABB()[1].Y}

	cHasVel := false
	rHasVel := false

	if cEnt != -1 {
		cHasVel = ecsContainer.Velocities.HasComponent(cEnt)
	}
	if rEnt != -1 {
		rHasVel = ecsContainer.Velocities.HasComponent(rEnt)
	}

	var cVel, rVel utils.Vec2
	if cHasVel {
		cVel, err = vm.GetLocalVector(cEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if rHasVel {
		rVel, err = vm.GetLocalVector(rEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for rectangle entity %d: %v\n", rEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	relVel := cVel.Subtract(rVel)

	if relVel.IsZero() {
		closestPoint := utils.Vec2{
			X: utils.Clamp(circleStart.X, rectMin.X, rectMax.X),
			Y: utils.Clamp(circleStart.Y, rectMin.Y, rectMax.Y),
		}
		collisionVector := circleStart.Subtract(closestPoint)
		distance := collisionVector.Length()
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 && distance != 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
		return utils.Vec2{X: 0, Y: 0}
	}

	// Expand rectangle by circle radius (Minkowski sum)
	expandedMin := rectMin.Subtract(utils.Vec2{X: cHit.GetRadius(), Y: cHit.GetRadius()})
	expandedMax := rectMax.Add(utils.Vec2{X: cHit.GetRadius(), Y: cHit.GetRadius()})

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
				return utils.Vec2{X: 0, Y: 0}
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
				return utils.Vec2{X: 0, Y: 0}
			}
		}
	}

	// Collision occurs at tEntry
	if tEntry >= 0 && tEntry <= 1 {
		contactPoint := circleStart.Add(relVel.Multiply(tEntry))
		closestPoint := utils.Vec2{
			X: utils.Clamp(contactPoint.X, rectMin.X, rectMax.X),
			Y: utils.Clamp(contactPoint.Y, rectMin.Y, rectMax.Y),
		}
		collisionVector := contactPoint.Subtract(closestPoint)
		distance := collisionVector.Length()
		if distance == 0 {
			// Touching edge, push out perpendicular to movement
			if math.Abs(relVel.X) > math.Abs(relVel.Y) {
				collisionVector = utils.Vec2{X: relVel.X, Y: 0}
			} else {
				collisionVector = utils.Vec2{X: 0, Y: relVel.Y}
			}
			if collisionVector.IsZero() {
				return utils.Vec2{X: 0, Y: 0}
			}
			return collisionVector.Normalized().Multiply(0.01) // Small nudge
		}
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
	}
	return utils.Vec2{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetRectangleRectangleCollision(
	r1Ent common.EntityId,
	r2Ent common.EntityId,
	r1Hit shapes.RectangleShape,
	r2Hit shapes.RectangleShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2 {
	var err error

	var r1WorldPos, r2WorldPos utils.Vec2

	if r1Ent != -1 {
		r1WorldPos, err = ecsContainer.TransformManager.GetWorldPos(r1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", r1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if r2Ent != -1 {
		r2WorldPos, err = ecsContainer.TransformManager.GetWorldPos(r2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for rectangle entity %d: %v\n", r2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	r1Min := utils.Vec2{X: r1WorldPos.X + r1Hit.GetAABB()[0].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[0].Y}
	r1Max := utils.Vec2{X: r1WorldPos.X + r1Hit.GetAABB()[1].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[1].Y}
	r2Min := utils.Vec2{X: r2WorldPos.X + r2Hit.GetAABB()[0].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[0].Y}
	r2Max := utils.Vec2{X: r2WorldPos.X + r2Hit.GetAABB()[1].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[1].Y}

	r1HasVel := false
	r2HasVel := false

	if r1Ent != -1 {
		r1HasVel = ecsContainer.Velocities.HasComponent(r1Ent)
	}
	if r2Ent != -1 {
		r2HasVel = ecsContainer.Velocities.HasComponent(r2Ent)
	}

	var r1Vel, r2Vel utils.Vec2
	if r1HasVel {
		r1Vel, err = ecsContainer.VelocityManager.GetLocalVector(r1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for rectangle entity %d: %v\n", r1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if r2HasVel {
		r2Vel, err = ecsContainer.VelocityManager.GetLocalVector(r2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for rectangle entity %d: %v\n", r2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	relVel := r1Vel.Subtract(r2Vel)

	if relVel.IsZero() {
		overlapX := math.Min(r1Max.X, r2Max.X) - math.Max(r1Min.X, r2Min.X)
		overlapY := math.Min(r1Max.Y, r2Max.Y) - math.Max(r1Min.Y, r2Min.Y)
		if overlapX > 0 && overlapY > 0 {
			if overlapX < overlapY {
				sign := 1.0
				if r1Min.X < r2Min.X {
					sign = -1.0
				}
				return utils.Vec2{X: overlapX * sign, Y: 0}
			} else {
				sign := 1.0
				if r1Min.Y < r2Min.Y {
					sign = -1.0
				}
				return utils.Vec2{X: 0, Y: overlapY * sign}
			}
		}
		return utils.Vec2{X: 0, Y: 0}
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
				return utils.Vec2{X: 0, Y: 0}
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
				return utils.Vec2{X: 0, Y: 0}
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
				return utils.Vec2{X: overlapX * sign, Y: 0}
			} else {
				sign := 1.0
				if relVel.Y < 0 {
					sign = -1.0
				}
				return utils.Vec2{X: 0, Y: overlapY * sign}
			}
		}
	}
	return utils.Vec2{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetCircleCircleCollision(
	c1Ent common.EntityId,
	c2Ent common.EntityId,
	c1Hit shapes.CircleShape,
	c2Hit shapes.CircleShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2 {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager
	var err error

	var c1WorldPos, c2WorldPos utils.Vec2

	if c1Ent != -1 {
		c1WorldPos, err = tm.GetWorldPos(c1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", c1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	if c2Ent != -1 {
		c2WorldPos, err = tm.GetWorldPos(c2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", c2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	center1 := utils.Vec2{X: c1WorldPos.X + c1Hit.GetOffset().X, Y: c1WorldPos.Y + c1Hit.GetOffset().Y}
	center2 := utils.Vec2{X: c2WorldPos.X + c2Hit.GetOffset().X, Y: c2WorldPos.Y + c2Hit.GetOffset().Y}

	c1HasVel := false
	c2HasVel := false

	if c1Ent != -1 {
		c1HasVel = ecsContainer.Velocities.HasComponent(c1Ent)
	}
	if c2Ent != -1 {
		c2HasVel = ecsContainer.Velocities.HasComponent(c2Ent)
	}

	var c1Vel, c2Vel utils.Vec2
	if c1HasVel {
		c1Vel, err = vm.GetWorldVector(c1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for circle entity %d: %v\n", c1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if c2HasVel {
		c2Vel, err = vm.GetWorldVector(c2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for circle entity %d: %v\n", c2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

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
		return utils.Vec2{X: penetrationDepth, Y: 0}
	}

	if relVel.IsZero() {
		return utils.Vec2{X: 0, Y: 0}
	}

	// 2. CCD: Sweep circle1 along relVel against circle2
	f := center1.Subtract(center2)

	a := relVel.Dot(relVel)
	b := 2 * f.Dot(relVel)
	c := f.Dot(f) - r*r

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return utils.Vec2{X: 0, Y: 0}
	}
	sqrtDisc := math.Sqrt(discriminant)

	// Since we handled initial overlap (c < 0), t1 is guaranteed to be the entry point
	t1 := (-b - sqrtDisc) / (2 * a)

	if t1 < 0 || t1 > 1 {
		return utils.Vec2{X: 0, Y: 0}
	}

	contact1 := center1.Add(relVel.Multiply(t1))
	collisionNormal := center2.Subtract(contact1)
	if collisionNormal.Length() == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}
	normal := collisionNormal.Normalized()

	// Penetration depth is the amount of movement remaining AFTER contact,
	// projected onto the contact normal. This handles pass-through correctly too.
	overshoot := relVel.Multiply(1 - t1)
	penetrationDepth := overshoot.Dot(normal)

	if penetrationDepth > 0 {
		return normal.Multiply(penetrationDepth)
	}

	return utils.Vec2{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetRectanglePolygonCollision(
	rEnt common.EntityId,
	pEnt common.EntityId,
	rHit shapes.RectangleShape,
	pHit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2 {
	rectAsPolygon, err := shapes.NewPolygonShape(
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

	return GetPolygonPolygonCollision(rEnt, pEnt, *rectAsPolygon, pHit, ecsContainer)
}

// Pass -1 in case of collision without entity
func GetCirclePolygonCollision(
	cEnt common.EntityId,
	pEnt common.EntityId,
	cHit shapes.CircleShape,
	pHit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2 {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager
	var err error

	var cWorldPos, pWorldPos utils.Vec2

	if cEnt != -1 {
		cWorldPos, err = tm.GetWorldPos(cEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	if pEnt != -1 {
		pWorldPos, err = tm.GetWorldPos(pEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", pEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	circleStart := utils.Vec2{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	polyVerts := GetWorldPolygonVertices(pHit, pWorldPos)

	cHasVel := false
	pHasVel := false

	if cEnt != -1 {
		cHasVel = ecsContainer.Velocities.HasComponent(cEnt)
	}
	if cEnt != -1 {
		pHasVel = ecsContainer.Velocities.HasComponent(pEnt)
	}

	var cVel, pVel utils.Vec2
	if cHasVel {
		cVel, err = vm.GetLocalVector(cEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for circle entity %d: %v\n", cEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if pHasVel {
		pVel, err = vm.GetLocalVector(pEnt, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for polygon entity %d: %v\n", pEnt, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	relVel := cVel.Subtract(pVel)

	if relVel.IsZero() {
		minDist := math.MaxFloat64
		var closest utils.Vec2
		for _, v := range polyVerts {
			d := v.Subtract(circleStart).Length()
			if d < minDist {
				minDist = d
				closest = v
			}
		}
		collisionVector := circleStart.Subtract(closest)
		distance := collisionVector.Length()
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 && distance != 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
		return utils.Vec2{X: 0, Y: 0}
	}

	// Swept test: move circle along relVel, check for intersection with polygon edges
	steps := 4
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		circlePos := circleStart.Add(relVel.Multiply(t))
		minDist := math.MaxFloat64
		var closest utils.Vec2
		for _, v := range polyVerts {
			d := v.Subtract(circlePos).Length()
			if d < minDist {
				minDist = d
				closest = v
			}
		}
		collisionVector := circlePos.Subtract(closest)
		distance := collisionVector.Length()
		penetrationDepth := cHit.GetRadius() - distance
		if penetrationDepth > 0 && distance != 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
	}
	return utils.Vec2{X: 0, Y: 0}
}

// Pass -1 in case of collision without entity
func GetPolygonPolygonCollision(
	p1Ent common.EntityId,
	p2Ent common.EntityId,
	p1Hit shapes.PolygonShape,
	p2Hit shapes.PolygonShape,
	ecsContainer *ecs.ECSContainer,
) utils.Vec2 {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager
	var err error

	var p1WorldPos, p2WorldPos utils.Vec2

	if p1Ent != -1 {
		p1WorldPos, err = tm.GetWorldPos(p1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", p1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	if p2Ent != -1 {
		p2WorldPos, err = tm.GetWorldPos(p2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting world position for polygon entity %d: %v\n", p2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	p1Verts := GetWorldPolygonVertices(p1Hit, p1WorldPos)
	p2Verts := GetWorldPolygonVertices(p2Hit, p2WorldPos)

	p1HasVel := false
	p2HasVel := false

	if p1Ent != -1 {
		p1HasVel = ecsContainer.Velocities.HasComponent(p1Ent)
	}
	if p2Ent != -1 {
		p2HasVel = ecsContainer.Velocities.HasComponent(p2Ent)
	}

	var p1Vel, p2Vel utils.Vec2
	if p1HasVel {
		p1Vel, err = vm.GetLocalVector(p1Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for polygon entity %d: %v\n", p1Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}
	if p2HasVel {
		p2Vel, err = vm.GetLocalVector(p2Ent, ecsContainer)
		if err != nil {
			log.Printf("Error getting velocity for polygon entity %d: %v\n", p2Ent, err)
			return utils.Vec2{X: 0, Y: 0}
		}
	}

	// Relative movement
	relVel := p1Vel.Subtract(p2Vel)
	if relVel.IsZero() {
		// Fallback to static SAT overlap check
		mtv, overlap := PolygonSAT(p1Verts, p2Verts)
		if overlap {
			return mtv
		}
		return utils.Vec2{X: 0, Y: 0}
	}

	// Swept SAT: conservative approach, move p1 along relVel and check for first overlap
	steps := 4
	minT := 1.1
	var mtv utils.Vec2
	found := false
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		offset := relVel.Multiply(t)
		movedVerts := make([]utils.Vec2, len(p1Verts))
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
	return utils.Vec2{X: 0, Y: 0}
}

func GetWorldPolygonVertices(p shapes.PolygonShape, worldPos utils.Vec2) []utils.Vec2 {
	verts := p.GetVertices()
	worldVerts := make([]utils.Vec2, len(verts))
	for i, v := range verts {
		worldVerts[i] = worldPos.Add(v)
	}
	return worldVerts
}

// PolygonSAT returns the minimum translation vector (MTV) and whether two convex polygons overlap.
// vertsA and vertsB must be in world space and ordered (clockwise or counterclockwise).
func PolygonSAT(vertsA, vertsB []utils.Vec2) (utils.Vec2, bool) {
	var mtv utils.Vec2
	minOverlap := math.MaxFloat64
	var smallestAxis utils.Vec2

	axes := getPolygonAxes(vertsA)
	axes = append(axes, getPolygonAxes(vertsB)...)

	for _, axis := range axes {
		minA, maxA := projectPolygon(axis, vertsA)
		minB, maxB := projectPolygon(axis, vertsB)

		overlap := math.Min(maxA, maxB) - math.Max(minA, minB)
		if overlap <= 0 {
			return utils.Vec2{X: 0, Y: 0}, false
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

func getPolygonAxes(verts []utils.Vec2) []utils.Vec2 {
	axes := make([]utils.Vec2, len(verts))
	for i := 0; i < len(verts); i++ {
		j := (i + 1) % len(verts)
		edge := verts[j].Subtract(verts[i])
		normal := utils.Vec2{X: -edge.Y, Y: edge.X}.Normalized()
		axes[i] = normal
	}
	return axes
}

func projectPolygon(axis utils.Vec2, verts []utils.Vec2) (float64, float64) {
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

func polygonCentroid(verts []utils.Vec2) utils.Vec2 {
	var c utils.Vec2
	for _, v := range verts {
		c = c.Add(v)
	}
	return c.Multiply(1.0 / float64(len(verts)))
}
