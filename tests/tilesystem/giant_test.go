package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

func diamond(t *testing.T, r float64) *shapes.PolygonShape {
	t.Helper()
	d, err := shapes.NewPolygonShape([]utils.Vec2f{{X: 0, Y: -r}, {X: r, Y: 0}, {X: 0, Y: r}, {X: -r, Y: 0}}, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func withBody(t *testing.T, s *testutil.Session, pos utils.Vec2f, shape shapes.Shape) common.EntityId {
	t.Helper()
	return testutil.AddMovable(t, s.ECS, pos, false, ecs.NewPhysicsColliderComponent(ecs.Layer_Enemy, ecs.Layer_Wall, ecs.Collider_Mob, shape))
}

func TestABodyTakesUpTheTilesItsShapeReachesIntoRatherThanTheBoxAroundIt(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	body := withBody(t, s, utils.TilePosToWorldPos(middle), diamond(t, 15))

	want := []utils.Vec2i{at(middle, 0, -1), at(middle, -1, 0), middle, at(middle, 1, 0), at(middle, 0, 1)}
	if got := s.ECS.Footprint(body); !slices.Equal(got, want) {
		t.Fatalf("the diamond takes up %v, want the cross of %v", got, want)
	}
	if s.ECS.IsEntityInTile(at(middle, 1, 1)) {
		t.Fatal("the diamond is counted as in the corner of the box around it")
	}
}

func TestABodyFitsWhereAllItWouldTakeUpIsWalkableAndEmpty(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	shape, err := templates.GiantSlimePhysicsShape()
	if err != nil {
		t.Fatal(err)
	}
	body := []shapes.Shape{shape}
	pos := utils.TilePosToWorldPos(middle)
	if !s.ECS.FitsAt(body, pos) {
		t.Fatal("a giant slime doesn't fit in the middle of open ground")
	}
	wall(t, s, at(middle, 1, 1))
	if s.ECS.FitsAt(body, pos) {
		t.Fatal("a giant slime fits with a wall where its body would be")
	}
	if !s.ECS.FitsAt(body, utils.TilePosToWorldPos(at(middle, -1, -1))) {
		t.Fatal("a giant slime doesn't fit away from the wall")
	}
	withBody(t, s, utils.TilePosToWorldPos(at(middle, -2, -1)), diamond(t, 4))
	if s.ECS.FitsAt(body, utils.TilePosToWorldPos(at(middle, -1, -1))) {
		t.Fatal("a giant slime fits on top of something")
	}
}

func TestAGiantSlimesHurtboxIsInsideItsHitboxAndItsFootprintSmaller(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	slime, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(doorYard(t, s, colonist)))
	if err != nil {
		t.Fatal(err)
	}
	hit, _ := s.ECS.HitboxColliders.Get(slime)
	hurt, _ := s.ECS.HurtboxColliders.Get(slime)
	physics, _ := s.ECS.PhysicsColliders.Get(slime)
	outline := hit.Shapes()[0].(*shapes.PolygonShape)
	for _, v := range hurt.Shapes()[0].(*shapes.PolygonShape).GetVertices() {
		if !shapes.OverlapsBox(outline, v.Subtract(utils.Vec2f{X: 0.5, Y: 0.5}), v.Add(utils.Vec2f{X: 0.5, Y: 0.5})) {
			t.Fatalf("the hurtbox's corner %v is outside the hitbox", v)
		}
	}
	box, footprint := shapes.BoundingAABB(hit.Shapes()), shapes.BoundingAABB(physics.Shapes())
	if footprint[0].X <= box[0].X || footprint[1].X >= box[1].X || footprint[0].Y <= box[0].Y || footprint[1].Y >= box[1].Y {
		t.Fatalf("the footprint %v isn't inside the hitbox %v", footprint, box)
	}
}

func TestAColliderTooBigForTheSpatialHashGridIsRefused(t *testing.T) {
	x := testutil.NewECS(t)
	huge, _ := shapes.NewCircleShape(data.SpatialHashGridCellSize, utils.Vec2f{})
	if _, err := x.AddNewEntity(ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewPhysicsColliderComponent(ecs.Layer_Enemy, ecs.Layer_Wall, ecs.Collider_Mob, huge)); err == nil {
		t.Fatal("an entity with a collider wider than a grid cell was added")
	}
}

func TestAGiantSlimeSlidesAlongAWallWithoutCatchingOnItsTiles(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner := storageSpot(t, s, colonist, 20, 16, 6)
	wall(t, s, rect(corner, at(corner, 15, 0))...)
	start := utils.TilePosToWorldPos(at(corner, 3, 2))
	slime, err := templates.SpawnGiantSlime(s.ECS, start)
	if err != nil {
		t.Fatal(err)
	}

	const ticks = 2 * data.TPS
	speed := utils.Vec2f{X: 1.5, Y: -0.75}
	for range ticks {
		if v, ok := s.ECS.Velocities.Get(slime); ok {
			v.SetLocalVector(speed)
		}
		liveColliding(t, s, 1, nil)
	}
	end := testutil.WorldPos(t, s.ECS, slime)
	wallBottom := utils.TilePosToWorldPos(corner).Y + data.TileSize/2
	if top := end.Y - 11; top < wallBottom-0.5 || top > wallBottom+1 {
		t.Fatalf("the slime's footprint is %v into the wall, want pressed against it", wallBottom-top)
	}
	if moved := end.X - start.X; moved < 0.8*speed.X*ticks {
		t.Fatalf("the slime slid %v along the wall in %d ticks, want close to %v", moved, ticks, speed.X*ticks)
	}
}
