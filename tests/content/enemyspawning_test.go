package content_test

import (
	"ebittest/content/enemyspawning"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// Loaded and passable everywhere, open to the sky where told, with walls where told
type spawnWorld struct {
	ecs.Terrain
	sky   bool
	walls bool
}

func (w spawnWorld) IsOpenToSky(utils.Vec2i) bool  { return w.sky }
func (w spawnWorld) IsOpaqueTile(utils.Vec2i) bool { return w.walls }
func (w spawnWorld) IsBlockedAt(utils.Vec2f) bool  { return w.walls }

func spawningPlayer(t *testing.T, world spawnWorld, tick uint64) (*ecs.ECSContainer, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	world.Terrain = x.Terrain()
	x.SetTerrain(world)
	x.TickIdx = tick
	player, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	return x, player
}

func spawned(x *ecs.ECSContainer) []common.EntityId {
	return x.SpawnedEnemies.GetEntities()
}

func spawn(t *testing.T, x *ecs.ECSContainer, player common.EntityId) {
	t.Helper()
	if err := enemyspawning.Tick(x, player); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, x)
}

const nightTick = data.DayTicks + data.DuskTicks

func TestEvilSlimesSpawnInTheDarkAwayFromThePlayerUpToTheCap(t *testing.T) {
	x, player := spawningPlayer(t, spawnWorld{}, 0)
	for range 100 {
		spawn(t, x, player)
		x.TickIdx += data.EnemySpawnIntervalTicks
	}

	if n := len(spawned(x)); n != data.MaxSpawnedEnemies {
		t.Fatalf("%d enemies spawned over 100 tries in the dark, want the cap of %d", n, data.MaxSpawnedEnemies)
	}
	for _, e := range spawned(x) {
		pos := testutil.WorldPos(t, x, e)
		tile := utils.WorldPosToTilePos(pos)
		if d := utils.Dist2(utils.Vec2i{}, tile); d < 64*64 || d > 128*128 {
			t.Errorf("an enemy spawned at %v, %v tiles² away, want 64 to 128 tiles", tile, d)
		}
		if pos != utils.TilePosToWorldPos(tile) {
			t.Errorf("an enemy spawned at %v, off its tile's center", pos)
		}
		if body, ok := x.Bodies.Get(e); !ok || body.Kind() != data.Body_Humanoid || !x.Inputs.HasComponent(e) {
			t.Errorf("entity %d spawned isn't an evil slime", e)
		}
	}
}

func TestEnemiesOnlySpawnEverySecondAndOnlyInTheDarkOnFreeGround(t *testing.T) {
	for _, c := range []struct {
		name  string
		world spawnWorld
		tick  uint64
	}{
		{"between tries", spawnWorld{}, data.EnemySpawnIntervalTicks + 1},
		{"in daylight", spawnWorld{sky: true}, 0},
		{"under the night sky", spawnWorld{sky: true}, nightTick},
		{"in walls", spawnWorld{walls: true}, 0},
	} {
		x, player := spawningPlayer(t, c.world, c.tick)
		for range 20 {
			spawn(t, x, player)
		}
		if n := len(spawned(x)); n != 0 {
			t.Errorf("%s: %d enemies spawned", c.name, n)
		}
	}
}

func TestEnemiesDontSpawnWhereThePlayerSeesOrSomethingStands(t *testing.T) {
	x, player := spawningPlayer(t, spawnWorld{}, 0)
	// Far-sighted enough to see the whole spawning ring in light that's dark enough to spawn in
	v, _ := x.Visions.Get(player)
	v.Update(utils.Vec2i{}, 1100, func(utils.Vec2i) bool { return false }, func(utils.Vec2i) uint8 { return 30 })
	for range 20 {
		spawn(t, x, player)
	}
	if n := len(spawned(x)); n != 0 {
		t.Fatalf("%d enemies spawned where the player sees", n)
	}

	y, other := spawningPlayer(t, spawnWorld{}, 0)
	for dy := -128; dy <= 128; dy++ {
		for dx := -128; dx <= 128; dx++ {
			if d := dx*dx + dy*dy; d >= 64*64 && d <= 128*128 {
				if _, err := y.SpawnItem(data.Item_Planks, utils.TilePosToWorldPos(utils.Vec2i{X: dx, Y: dy})); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	spawn(t, y, other)
	if n := len(spawned(y)); n != 0 {
		t.Fatalf("%d enemies spawned where items lie", n)
	}
}

func TestSpawnedEnemiesTooFarAwayAreRemovedButOthersStay(t *testing.T) {
	x, player := spawningPlayer(t, spawnWorld{walls: true}, 1)
	add := func(tile utils.Vec2i, marked bool) common.EntityId {
		t.Helper()
		e, err := templates.SpawnEvilSlime(x, utils.TilePosToWorldPos(tile))
		if err != nil {
			t.Fatal(err)
		}
		if marked {
			x.AddComponent(e, ecs.NewSpawnedEnemyComponent())
		}
		return e
	}
	far := add(utils.Vec2i{X: 129}, true)
	near := add(utils.Vec2i{X: 90, Y: 90}, true)
	placed := add(utils.Vec2i{Y: -200}, false)

	spawn(t, x, player)
	if x.HasEntity(far) {
		t.Fatal("a spawned enemy 129 tiles away wasn't removed")
	}
	if !x.HasEntity(near) || !x.HasEntity(placed) {
		t.Fatal("a spawned enemy within 128 tiles, or an enemy that wasn't spawned, was removed")
	}
}
