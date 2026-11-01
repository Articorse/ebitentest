package collisionsystem

import (
	"cmp"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"log"
	"slices"
)

// Keeps the order of the pairs, with each pair's A taken from aColliders and B from bColliders
func GetCollisions(
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	potentialCollisions []common.EntityPair,
	ecsContainer *ecs.ECSContainer,
) []common.Collision {
	collisions := []common.Collision{}
	for _, pair := range potentialCollisions {
		aCol, aOk := aColliders.Collider(pair.A)
		bCol, bOk := bColliders.Collider(pair.B)
		if !aOk || !bOk {
			continue
		}

		if c, ok := firstShapeCollision(pair, aCol.Shapes(), bCol.Shapes(), ecsContainer); ok {
			collisions = append(collisions, c)
		}
	}

	return collisions
}

func firstShapeCollision(
	pair common.EntityPair,
	aShapes []shapes.Shape,
	bShapes []shapes.Shape,
	ecsContainer *ecs.ECSContainer,
) (common.Collision, bool) {
	for aIdx, aShape := range aShapes {
		for bIdx, bShape := range bShapes {
			v := ShapeCollision(pair.A, pair.B, aShape, bShape, ecsContainer)
			if !v.IsZero() {
				return common.Collision{A: pair.A, B: pair.B, Vector: v, AShapeIdx: aIdx, BShapeIdx: bIdx}, true
			}
		}
	}

	return common.Collision{}, false
}

// The vector is how far B has to move out of A, or zero without a collision. Pass -1 for a shape without an entity.
func ShapeCollision(eA, eB common.EntityId, aShape, bShape shapes.Shape, ecsContainer *ecs.ECSContainer) utils.Vec2f {
	switch aS := aShape.(type) {
	case *shapes.RectangleShape:
		switch bS := bShape.(type) {
		case *shapes.RectangleShape:
			return GetRectangleRectangleCollision(eA, eB, *aS, *bS, ecsContainer)
		case *shapes.CircleShape:
			return GetRectangleCircleCollision(eA, eB, *aS, *bS, ecsContainer)
		case *shapes.PolygonShape:
			return GetRectanglePolygonCollision(eA, eB, *aS, *bS, ecsContainer)
		}
	case *shapes.CircleShape:
		switch bS := bShape.(type) {
		case *shapes.RectangleShape:
			return GetRectangleCircleCollision(eB, eA, *bS, *aS, ecsContainer).Multiply(-1)
		case *shapes.CircleShape:
			return GetCircleCircleCollision(eA, eB, *aS, *bS, ecsContainer)
		case *shapes.PolygonShape:
			return GetCirclePolygonCollision(eA, eB, *aS, *bS, ecsContainer)
		}
	case *shapes.PolygonShape:
		switch bS := bShape.(type) {
		case *shapes.RectangleShape:
			return GetRectanglePolygonCollision(eB, eA, *bS, *aS, ecsContainer).Multiply(-1)
		case *shapes.CircleShape:
			return GetCirclePolygonCollision(eB, eA, *bS, *aS, ecsContainer).Multiply(-1)
		case *shapes.PolygonShape:
			return GetPolygonPolygonCollision(eA, eB, *aS, *bS, ecsContainer)
		}
	}

	log.Printf("unsupported collider shape types for collision detection: %T and %T", aShape, bShape)
	return utils.Vec2f{}
}

// For colliders of a single kind, where a pair collides the same either way round, so is listed once as given
func GetAABBCollisions(
	colliders ecs.ColliderSet,
	proximateEntities []common.EntityPair,
	ecsContainer *ecs.ECSContainer,
) []common.EntityPair {
	collisions := []common.EntityPair{}
	for _, pair := range proximateEntities {
		if aabbsCollide(pair, colliders, colliders, ecsContainer) {
			collisions = append(collisions, pair)
		}
	}

	return collisions
}

// For pairs where A acts on B, such as a hurtbox damaging a hitbox, so each pair is tried both ways round
func GetDirectedAABBCollisions(
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	proximateEntities []common.EntityPair,
	ecsContainer *ecs.ECSContainer,
) []common.EntityPair {
	collisions := []common.EntityPair{}
	for _, pair := range proximateEntities {
		for _, p := range []common.EntityPair{pair, {A: pair.B, B: pair.A}} {
			if aabbsCollide(p, aColliders, bColliders, ecsContainer) {
				collisions = append(collisions, p)
			}
		}
	}

	slices.SortFunc(collisions, func(p, q common.EntityPair) int {
		return cmp.Or(cmp.Compare(p.A, q.A), cmp.Compare(p.B, q.B))
	})

	return collisions
}

func aabbsCollide(
	pair common.EntityPair,
	aColliders ecs.ColliderSet,
	bColliders ecs.ColliderSet,
	ecsContainer *ecs.ECSContainer,
) bool {
	aCol, aOk := aColliders.Collider(pair.A)
	bCol, bOk := bColliders.Collider(pair.B)
	if !aOk || !bOk || !aCol.Enabled() || !bCol.Enabled() {
		return false
	}

	if aCol.Layer()&bCol.Mask() == 0 || bCol.Layer()&aCol.Mask() == 0 {
		return false
	}

	aAABB, err := ecsContainer.WorldAABB(pair.A, aCol.LocalPaddedAABB())
	if err != nil {
		log.Printf("Error getting AABB for entity %d: %v\n", pair.A, err)
		return false
	}

	bAABB, err := ecsContainer.WorldAABB(pair.B, bCol.LocalPaddedAABB())
	if err != nil {
		log.Printf("Error getting AABB for entity %d: %v\n", pair.B, err)
		return false
	}

	return utils.DetectAABBCollision(aAABB, bAABB)
}
