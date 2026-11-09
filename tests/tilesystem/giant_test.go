package tilesystem_test

import (
	"ebittest/content/inputsources"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/pathfinding"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"strings"
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

// Open ground whose bottom half is walled in, with a gap a tile wide at x 3 along the top of the walls and one three
// tiles wide at x 10 to 12, both from the top-left corner. Returns the tiles above and below the narrow gap, 3 tiles
// from it.
func wallWithGaps(t *testing.T, s *testutil.Session, colonist common.EntityId) (corner, above, below utils.Vec2i) {
	t.Helper()
	corner = storageSpot(t, s, colonist, 20, 16, 10)
	for x := 0; x < 16; x++ {
		if x != 3 && (x < 10 || x > 12) {
			wall(t, s, at(corner, x, 4))
		}
		wall(t, s, at(corner, x, 9))
	}
	for y := 5; y < 9; y++ {
		wall(t, s, at(corner, 0, y), at(corner, 15, y))
	}
	return corner, at(corner, 3, 1), at(corner, 3, 7)
}

func TestAGiantSlimesPathsKeepRoomForItsBody(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner, above, below := wallWithGaps(t, s, colonist)
	giant, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(above))
	if err != nil {
		t.Fatal(err)
	}

	path, ok := pathfinding.Find(above, below, data.PathMaxExpandedTiles, s.ECS.WalkSpeedOf(giant))
	if !ok {
		t.Fatal("the giant slime found no way round the wall")
	}
	if slices.Contains(path, at(corner, 3, 4)) || !slices.Contains(path, at(corner, 11, 4)) {
		t.Fatalf("the giant slime's path is %v, want it through the middle of the wide gap", path)
	}
	small, ok := pathfinding.Find(above, below, data.PathMaxExpandedTiles, s.ECS.WalkSpeedOf(colonist))
	if !ok || !slices.Contains(small, at(corner, 3, 4)) {
		t.Fatalf("a colonist's path is %v, want it through the narrow gap", small)
	}
}

func TestAGiantSlimeDoesntHeadStraightForAGapItCantFitThrough(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	_, above, below := wallWithGaps(t, s, colonist)
	giant, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(above))
	if err != nil {
		t.Fatal(err)
	}
	in, _ := s.ECS.Inputs.Get(giant)
	p := in.Params().(*inputsources.InputRusherParams)
	p.State, p.LastSeen = inputsources.Rusher_Search, below

	input, err := inputsources.RusherInputSource(giant, s.ECS.TickIdx, p, s.ECS)
	if err != nil {
		t.Fatal(err)
	}
	if input.Analog1X < 0.5 {
		t.Fatalf("the giant slime heads %v, %v, want towards the wide gap to its right", input.Analog1X, input.Analog1Y)
	}
}

func TestAGiantSlimeCantWalkWhereAWallIsInItsWay(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	giant, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(at(middle, 0, -3)))
	if err != nil {
		t.Fatal(err)
	}
	speed := s.ECS.WalkSpeedOf(giant)
	for _, c := range []struct {
		name string
		wall utils.Vec2i
		want bool
	}{
		{"above", at(middle, 0, -1), false},
		{"below", at(middle, 0, 1), false},
		{"to the left", at(middle, -1, 0), false},
		{"diagonally", at(middle, 1, 1), false},
		{"two tiles up", at(middle, 0, -2), true},
		{"two tiles to the right", at(middle, 2, 0), true},
	} {
		wall(t, s, c.wall)
		if walkable := speed(middle) > 0; walkable != c.want {
			t.Errorf("with a wall %s, the giant slime can walk on the tile %v, want %v", c.name, walkable, c.want)
		}
		if destroyed, _ := s.Chunks.DamageTile(c.wall, 1000); !destroyed {
			t.Fatal("couldn't take the wall down")
		}
	}
}

func TestWhereABodyFitsFollowsWhatsBuiltAndGrownAcrossChunkEdges(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	start := colonistTile(t, s, colonist)
	edge := (start.X/data.ChunkSize + 1) * data.ChunkSize
	lo, hi := utils.Vec2i{X: -1, Y: -1}, utils.Vec2i{X: 1, Y: 1}
	for y := start.Y - 40; y <= start.Y+40; y++ {
		inside, across := utils.Vec2i{X: edge - 1, Y: y}, utils.Vec2i{X: edge, Y: y}
		if speed, _ := s.Chunks.BodySpeeds(lo, hi)(inside); speed <= 0 || !s.Chunks.CanPlaceTile(across) || !s.Chunks.CanPlaceTile(at(across, 0, 1)) {
			continue
		}
		wall(t, s, across)
		if speed, _ := s.Chunks.BodySpeeds(lo, hi)(inside); speed > 0 {
			t.Fatal("a body fits where a wall was built across the chunk edge")
		}
		if destroyed, _ := s.Chunks.DamageTile(across, 1000); !destroyed {
			t.Fatal("couldn't take the wall down")
		}
		if speed, _ := s.Chunks.BodySpeeds(lo, hi)(inside); speed <= 0 {
			t.Fatal("a body doesn't fit once the wall across the chunk edge is gone")
		}
		if !s.Chunks.PlaceFeature(at(across, 0, 1), data.Feature_Tree) {
			t.Fatal("couldn't plant a tree")
		}
		if speed, _ := s.Chunks.BodySpeeds(lo, hi)(inside); speed > 0 {
			t.Fatal("a body fits where a tree grew across the chunk edge")
		}
		return
	}
	t.Fatal("no room for a body by the chunk edge")
}

func TestAGiantSlimeReachesOnlyWhatItsBodyFitsItsWayTo(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner, above, below := wallWithGaps(t, s, colonist)
	giant, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(above))
	if err != nil {
		t.Fatal(err)
	}
	if !s.ECS.Reachable(giant, above, below) {
		t.Fatal("the giant slime can't reach through the wide gap")
	}
	if !s.ECS.Reachable(giant, above, at(corner, 3, 4)) {
		t.Fatal("the giant slime can't reach what stands in the narrow gap, where it can't stand itself")
	}

	wall(t, s, at(corner, 10, 4), at(corner, 12, 4))
	if s.ECS.Reachable(giant, above, below) {
		t.Fatal("the giant slime reaches past a wall whose gaps it doesn't fit through")
	}
	if !s.ECS.Reachable(colonist, above, below) {
		t.Fatal("a colonist can't reach through the gaps")
	}
	if closest, ok := s.ECS.ClosestReachable(giant, above, at(corner, 3, 5)); !ok || closest != at(corner, 3, 2) {
		t.Fatalf("the closest the giant slime can get is %v (%v), want %v, right up to the narrow gap", closest, ok, at(corner, 3, 2))
	}
	if lines := s.Chunks.RegionDebugLines(above); !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "3x3 Region") }) {
		t.Fatalf("the tooltip shows %v, without the giant slime's regions", lines)
	}
}

func TestAGiantSlimeSearchingWhereItCantFitGoesAsCloseAsItCan(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	corner, _, _ := wallWithGaps(t, s, colonist)
	wall(t, s, at(corner, 10, 4), at(corner, 12, 4))
	s.ECS.SetDisabled(colonist, true)
	giant, err := templates.SpawnGiantSlime(s.ECS, utils.TilePosToWorldPos(at(corner, 12, 1)))
	if err != nil {
		t.Fatal(err)
	}
	in, _ := s.ECS.Inputs.Get(giant)
	p := in.Params().(*inputsources.InputRusherParams)
	p.State, p.LastSeen = inputsources.Rusher_Search, at(corner, 3, 5)

	if !live(t, s, 30*data.TPS, func() bool { return p.State == inputsources.Rusher_Idle }) {
		t.Fatalf("the giant slime is still %v, on %v", p.State, utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, giant)))
	}
	if at := utils.WorldPosToTilePos(testutil.WorldPos(t, s.ECS, giant)); at != (utils.Vec2i{X: corner.X + 3, Y: corner.Y + 2}) {
		t.Fatalf("the giant slime gave up on %v, want as close as it can get, by the narrow gap", at)
	}
}
