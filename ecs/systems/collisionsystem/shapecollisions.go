package collisionsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/collidershapes"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"math"
)

func getRectangleCircleCollision(
	rEnt common.EntityId,
	cEnt common.EntityId,
	rHit collidershapes.RectangleShape,
	cHit collidershapes.CircleShape,
	world *ecs.World,
) utils.Vec2 {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	cWorldPos, err := tm.GetWorldPos(cEnt, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	rWorldPos, err := tm.GetWorldPos(rEnt, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", rEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	circleStart := utils.Vec2{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	rectMin := utils.Vec2{X: rWorldPos.X + rHit.GetAABB()[0].X, Y: rWorldPos.Y + rHit.GetAABB()[0].Y}
	rectMax := utils.Vec2{X: rWorldPos.X + rHit.GetAABB()[1].X, Y: rWorldPos.Y + rHit.GetAABB()[1].Y}

	cVel, err := vm.GetLocalVector(cEnt, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	rVel, err := vm.GetLocalVector(rEnt, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for rectangle entity %d: %v\n", rEnt, err)
		return utils.Vec2{X: 0, Y: 0}
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

func getRectangleRectangleCollision(
	r1Ent common.EntityId,
	r2Ent common.EntityId,
	r1Hit collidershapes.RectangleShape,
	r2Hit collidershapes.RectangleShape,
	world *ecs.World,
) utils.Vec2 {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	r1WorldPos, err := tm.GetWorldPos(r1Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", r1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	r2WorldPos, err := tm.GetWorldPos(r2Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for rectangle entity %d: %v\n", r2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	r1Min := utils.Vec2{X: r1WorldPos.X + r1Hit.GetAABB()[0].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[0].Y}
	r1Max := utils.Vec2{X: r1WorldPos.X + r1Hit.GetAABB()[1].X, Y: r1WorldPos.Y + r1Hit.GetAABB()[1].Y}
	r2Min := utils.Vec2{X: r2WorldPos.X + r2Hit.GetAABB()[0].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[0].Y}
	r2Max := utils.Vec2{X: r2WorldPos.X + r2Hit.GetAABB()[1].X, Y: r2WorldPos.Y + r2Hit.GetAABB()[1].Y}

	r1Vel, err := vm.GetLocalVector(r1Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for rectangle entity %d: %v\n", r1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	r2Vel, err := vm.GetLocalVector(r2Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for rectangle entity %d: %v\n", r2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
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

func getCircleCircleCollision(
	c1Ent common.EntityId,
	c2Ent common.EntityId,
	c1Hit collidershapes.CircleShape,
	c2Hit collidershapes.CircleShape,
	world *ecs.World,
) utils.Vec2 {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	c1WorldPos, err := tm.GetWorldPos(c1Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", c1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	c2WorldPos, err := tm.GetWorldPos(c2Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", c2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	center1 := utils.Vec2{X: c1WorldPos.X + c1Hit.GetOffset().X, Y: c1WorldPos.Y + c1Hit.GetOffset().Y}
	center2 := utils.Vec2{X: c2WorldPos.X + c2Hit.GetOffset().X, Y: c2WorldPos.Y + c2Hit.GetOffset().Y}

	c1Vel, err := vm.GetLocalVector(c1Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for circle entity %d: %v\n", c1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	c2Vel, err := vm.GetLocalVector(c2Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for circle entity %d: %v\n", c2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	relVel := c1Vel.Subtract(c2Vel)

	if relVel.IsZero() {
		collisionVector := center2.Subtract(center1)
		distance := collisionVector.Length()
		penetrationDepth := c1Hit.GetRadius() + c2Hit.GetRadius() - distance
		if penetrationDepth > 0 && distance != 0 {
			return collisionVector.Normalized().Multiply(penetrationDepth)
		}
		return utils.Vec2{X: 0, Y: 0}
	}

	// CCD: Sweep circle1 along relVel against circle2
	s := center1
	f := s.Subtract(center2)
	r := c1Hit.GetRadius() + c2Hit.GetRadius()

	a := relVel.Dot(relVel)
	b := 2 * f.Dot(relVel)
	c := f.Dot(f) - r*r

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return utils.Vec2{X: 0, Y: 0}
	}
	sqrtDisc := math.Sqrt(discriminant)
	t1 := (-b - sqrtDisc) / (2 * a)
	t2 := (-b + sqrtDisc) / (2 * a)

	var t float64 = -1
	if t1 >= 0 && t1 <= 1 {
		t = t1
	} else if t2 >= 0 && t2 <= 1 {
		t = t2
	}
	if t < 0 {
		return utils.Vec2{X: 0, Y: 0}
	}

	contact1 := s.Add(relVel.Multiply(t))
	contact2 := center2
	collisionNormal := contact2.Subtract(contact1)
	distance := collisionNormal.Length()
	if distance == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}
	penetrationDepth := r - distance
	if penetrationDepth > 0 {
		return collisionNormal.Normalized().Multiply(penetrationDepth)
	}
	return utils.Vec2{X: 0, Y: 0}
}

func getRectanglePolygonCollision(
	rEnt common.EntityId,
	pEnt common.EntityId,
	rHit collidershapes.RectangleShape,
	pHit collidershapes.PolygonShape,
	world *ecs.World,
) utils.Vec2 {
	rectAsPolygon, err := collidershapes.NewPolygonShape(
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

	return getPolygonPolygonCollision(rEnt, pEnt, *rectAsPolygon, pHit, world)
}

func getCirclePolygonCollision(
	cEnt common.EntityId,
	pEnt common.EntityId,
	cHit collidershapes.CircleShape,
	pHit collidershapes.PolygonShape,
	world *ecs.World,
) utils.Vec2 {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	cWorldPos, err := tm.GetWorldPos(cEnt, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	pWorldPos, err := tm.GetWorldPos(pEnt, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", pEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}

	circleStart := utils.Vec2{X: cWorldPos.X + cHit.GetOffset().X, Y: cWorldPos.Y + cHit.GetOffset().Y}
	polyVerts := GetWorldPolygonVertices(pHit, pWorldPos)

	cVel, err := vm.GetLocalVector(cEnt, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for circle entity %d: %v\n", cEnt, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	pVel, err := vm.GetLocalVector(pEnt, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for polygon entity %d: %v\n", pEnt, err)
		return utils.Vec2{X: 0, Y: 0}
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

func getPolygonPolygonCollision(
	p1Ent common.EntityId,
	p2Ent common.EntityId,
	p1Hit collidershapes.PolygonShape,
	p2Hit collidershapes.PolygonShape,
	world *ecs.World,
) utils.Vec2 {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	// Get world positions and transformed vertices
	p1WorldPos, err := tm.GetWorldPos(p1Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", p1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	p2WorldPos, err := tm.GetWorldPos(p2Ent, world.Transforms, world.Parents)
	if err != nil {
		log.Printf("Error getting world position for polygon entity %d: %v\n", p2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	p1Verts := GetWorldPolygonVertices(p1Hit, p1WorldPos)
	p2Verts := GetWorldPolygonVertices(p2Hit, p2WorldPos)

	// Get velocities
	p1Vel, err := vm.GetLocalVector(p1Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for polygon entity %d: %v\n", p1Ent, err)
		return utils.Vec2{X: 0, Y: 0}
	}
	p2Vel, err := vm.GetLocalVector(p2Ent, world.Velocities)
	if err != nil {
		log.Printf("Error getting velocity for polygon entity %d: %v\n", p2Ent, err)
		return utils.Vec2{X: 0, Y: 0}
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

func GetWorldPolygonVertices(p collidershapes.PolygonShape, worldPos utils.Vec2) []utils.Vec2 {
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
