package collisionsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/components/hitboxes"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"log"
	"slices"
)

func ResolveCollisions(
	collisions map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
	velocities map[ecscommon.EntityId]*components.Velocity,
) (collisionsResolved uint64, err error) {
	for eA, cols := range collisions {
		for eB, colVector := range cols {
			colA, ok := colliders[eA]
			if !ok {
				return collisionsResolved, fmt.Errorf("colliding entity has no collider: ", eA)
			}
			colB, ok := colliders[eB]
			if !ok {
				return collisionsResolved, fmt.Errorf("colliding entity has no collider: ", eB)
			}

			traA, ok := transforms[eA]
			if !ok {
				return collisionsResolved, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}
			}
			traB, ok := transforms[eB]
			if !ok {
				return collisionsResolved, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eB,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}
			}

			velA, ok := velocities[eA]
			if !ok {
				continue
			}
			velB, ok := velocities[eB]
			if !ok {
				continue
			}

			var mobTra *components.Transform
			var mobVel *components.Velocity
			var staticVel *components.Velocity

			if colA.Type == components.Mob && colB.Type == components.Static {
				mobTra = traA
				mobVel = velA
				staticVel = velB
			} else if colB.Type == components.Mob && colA.Type == components.Static {
				colVector = colVector.Multiply(-1)
				mobTra = traB
				mobVel = velB
				staticVel = velA
			} else {
				continue
			}

			mobTra.SetPos(mobTra.GetPos().Add(colVector))

			normal := colVector.Normalized()
			relativeVelocity := mobVel.Vector.Subtract(staticVel.Vector)
			velocityAlongNormal := relativeVelocity.Dot(normal)

			if velocityAlongNormal < 0 {
				restitution := data.Bounciness
				impulseMagnitude := -(1 + restitution) * velocityAlongNormal
				impulse := normal.Multiply(impulseMagnitude)
				mobVel.Vector = mobVel.Vector.Add(impulse)
			}

			collisionsResolved++
		}
	}

	return collisionsResolved, nil
}

func GetCollisions(
	potentialCollisions map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2, error) {
	collisions := make(map[ecscommon.EntityId]map[ecscommon.EntityId]utils.Vec2)

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
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eB,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}
			}

			if eA == eB {
				continue
			}

			if collidedEntities, ok := collisions[eB]; ok {
				if _, ok := collidedEntities[eA]; ok {
					continue
				}
			}

			var collisionVector utils.Vec2
			var aCollidedHitbox hitboxes.Hitbox
			var bCollidedHitbox hitboxes.Hitbox

			for _, aHitbox := range colA.Hitboxes {
				for _, bHitbox := range colB.Hitboxes {
					switch aH := aHitbox.(type) {
					case *hitboxes.RectangleHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectangleRectangleCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.CircleHitbox:
							collisionVector = getRectangleCircleCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.PolygonHitbox:
							collisionVector = getRectanglePolygonCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
						}
					case *hitboxes.CircleHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectangleCircleCollision(*bH, *aH, *colB, *colA, *traB, *traA)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.CircleHitbox:
							collisionVector = getCircleCircleCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						case *hitboxes.PolygonHitbox:
							collisionVector = getCirclePolygonCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
						}
					case *hitboxes.PolygonHitbox:
						switch bH := bHitbox.(type) {
						case *hitboxes.RectangleHitbox:
							collisionVector = getRectanglePolygonCollision(*bH, *aH, *colB, *colA, *traB, *traA)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.CircleHitbox:
							collisionVector = getCirclePolygonCollision(*bH, *aH, *colB, *colA, *traB, *traA)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
							collisionVector = collisionVector.Multiply(-1)
						case *hitboxes.PolygonHitbox:
							collisionVector = getPolygonPolygonCollision(*aH, *bH, *colA, *colB, *traA, *traB)
							aCollidedHitbox = aHitbox
							bCollidedHitbox = bHitbox
						default:
							log.Printf("unsupported hitbox type for collision detection: %T", bH)
						}
					}
				}
			}

			prevRelativePosVector := traA.GetPrevPos().Add(aCollidedHitbox.GetOffset()).Subtract(traB.GetPrevPos().Add(bCollidedHitbox.GetOffset()))
			if prevRelativePosVector.Dot(collisionVector) < 0 {
				collisionVector = collisionVector.Multiply(-1)
			}

			if !collisionVector.IsZero() {
				if _, ok := collisions[eA]; !ok {
					collisions[eA] = make(map[ecscommon.EntityId]utils.Vec2)
				}
				collisions[eA][eB] = collisionVector
			}
		}
	}

	return collisions, nil
}

func GetAABBCollisions(
	proximateEntities map[ecscommon.EntityId][]ecscommon.EntityId,
	colliders map[ecscommon.EntityId]*components.Collider,
	transforms map[ecscommon.EntityId]*components.Transform,
) (map[ecscommon.EntityId][]ecscommon.EntityId, error) {
	collisions := make(map[ecscommon.EntityId][]ecscommon.EntityId)

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
				return nil, &ecscommon.ErrorMissingComponentDependency{
					Entity:           eA,
					PresentComponent: "Collider",
					MissingComponent: "Transform",
				}

			}
			traB, ok := transforms[eB]
			if !ok {
				return nil, &ecscommon.ErrorMissingComponentDependency{
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

			a := [2]utils.Vec2{
				utils.Vec2{X: traA.GetPos().X + colA.GetAABB()[0].X, Y: traA.GetPos().Y + colA.GetAABB()[0].Y},
				utils.Vec2{X: traA.GetPos().X + colA.GetAABB()[1].X, Y: traA.GetPos().Y + colA.GetAABB()[1].Y},
			}
			b := [2]utils.Vec2{
				utils.Vec2{X: traB.GetPos().X + colB.GetAABB()[0].X, Y: traB.GetPos().Y + colB.GetAABB()[0].Y},
				utils.Vec2{X: traB.GetPos().X + colB.GetAABB()[1].X, Y: traB.GetPos().Y + colB.GetAABB()[1].Y},
			}

			if detectAABBCollision(a, b) {
				v, ok := collisions[eA]
				if !ok {
					collisions[eA] = []ecscommon.EntityId{eB}
				}
				collisions[eA] = append(v, eB)
			}
		}
	}

	return collisions, nil
}

func detectAABBCollision(a, b [2]utils.Vec2) bool {
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
