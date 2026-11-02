package content_test

import (
	"ebittest/content/abilities"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"maps"
	"testing"
)

// A holder at the origin with the given number of planks in its first slot, in hand
func newPlankHolder(t *testing.T, terrain ecs.Terrain, planks int) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	x.SetTerrain(terrain)

	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false,
		ecs.NewInventoryComponent(2),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}))
	for range planks {
		plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := x.PickUpItem(holder, plank); err != nil || !ok {
			t.Fatalf("picking up a plank: %v, %v", ok, err)
		}
		testutil.RemoveScheduled(t, x)
	}

	inv, _ := x.Inventories.Get(holder)
	stack, _ := inv.Slot(0)
	return x, holder, stack
}

func place(t *testing.T, x *ecs.ECSContainer, holder common.EntityId, tile utils.Vec2i) bool {
	t.Helper()
	return strike(t, x, holder, tile)
}

func addBlocker(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f) {
	t.Helper()
	shape, err := shapes.NewRectangleShape(6, 6, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	testutil.AddMovable(t, x, pos, false, ecs.NewPhysicsColliderComponent(1, 1, ecs.Collider_Mob, shape))
}

func TestPlanksArePlacedOnlyOnEmptyGroundInReachAndSight(t *testing.T) {
	half := float64(data.TileSize) / 2
	cases := []struct {
		name     string
		rock     []utils.Vec2i
		trees    []utils.Vec2i
		blockers []utils.Vec2f
		items    []utils.Vec2f
		aim      utils.Vec2i
		placed   bool
	}{
		{"empty ground", nil, nil, nil, nil, utils.Vec2i{X: 2}, true},
		{"empty ground at the edge of reach", nil, nil, nil, nil, utils.Vec2i{X: 3}, true},
		{"ground out of reach", nil, nil, nil, nil, utils.Vec2i{X: 4}, false},
		{"diagonally out of reach", nil, nil, nil, nil, utils.Vec2i{X: 3, Y: 1}, false},
		{"rock", []utils.Vec2i{{X: 2}}, nil, nil, nil, utils.Vec2i{X: 2}, false},
		{"ground behind rock", []utils.Vec2i{{X: 1}}, nil, nil, nil, utils.Vec2i{X: 2}, false},
		{"a tree", nil, []utils.Vec2i{{X: 2}}, nil, nil, utils.Vec2i{X: 2}, false},
		{"ground with something standing on it", nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 2})}, nil, utils.Vec2i{X: 2}, false},
		{"ground something reaches into from the next tile", nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 3}).Subtract(utils.Vec2f{X: half})}, nil, utils.Vec2i{X: 2}, false},
		{"ground something stands flush against", nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 3}).Add(utils.Vec2f{X: 3 - half})}, nil, utils.Vec2i{X: 2}, true},
		{"ground with an item on it", nil, nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 2})}, utils.Vec2i{X: 2}, false},
	}

	for _, c := range cases {
		terrain := newFakeTerrain(c.rock...)
		for _, tree := range c.trees {
			terrain.trees[tree] = true
		}
		x, holder, _ := newPlankHolder(t, terrain, 2)
		for _, pos := range c.blockers {
			addBlocker(t, x, pos)
		}
		for _, pos := range c.items {
			if _, err := x.SpawnItem(data.Item_Planks, pos); err != nil {
				t.Fatal(err)
			}
		}

		targeting, ok := x.HeldTileTargeting(holder)
		if !ok {
			t.Fatal("planks in hand don't target tiles")
		}
		tile, targeted := targeting.TargetTile(utils.Vec2f{}, utils.TilePosToWorldPos(c.aim), x)
		if targeted != c.placed || (targeted && tile != c.aim) {
			t.Errorf("%s: targets %v, %v; want %v", c.name, tile, targeted, c.placed)
		}

		want := map[utils.Vec2i]data.TileEnum{}
		if c.placed {
			want[c.aim] = data.Tile_Planks
		}
		if used := place(t, x, holder, c.aim); used != c.placed || !maps.Equal(terrain.placed, want) {
			t.Errorf("%s: placing used %v and placed %v, want %v", c.name, used, terrain.placed, want)
		}
	}
}

func TestTheSpotTheHolderStandsOnIsTaken(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder, _ := newPlankHolder(t, terrain, 1)
	shape, err := shapes.NewRectangleShape(6, 6, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	x.AddComponent(holder, ecs.NewPhysicsColliderComponent(1, 1, ecs.Collider_Mob, shape))

	if place(t, x, holder, utils.Vec2i{}) || len(terrain.placed) != 0 {
		t.Fatal("a tile was placed on whoever placed it")
	}
}

func TestEachPlacementUsesUpAPlankAndTheLastEmptiesTheSlot(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder, stack := newPlankHolder(t, terrain, 2)
	tick := func(ms int) {
		t.Helper()
		for range ms / data.TickMs {
			if err := abilitysystem.Tick(x); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !place(t, x, holder, utils.Vec2i{X: 1}) {
		t.Fatal("the first plank wasn't placed")
	}
	if item, _ := x.Items.Get(stack); item.Count() != 1 {
		t.Fatalf("%d planks are left after placing one of 2, want 1", item.Count())
	}

	tick(240)
	if place(t, x, holder, utils.Vec2i{X: -1}) {
		t.Fatal("a plank was placed before the 250ms cooldown ran out")
	}
	tick(40)
	if !place(t, x, holder, utils.Vec2i{X: -1}) {
		t.Fatal("the last plank wasn't placed after the cooldown")
	}

	inv, _ := x.Inventories.Get(holder)
	if _, ok := inv.Slot(0); ok {
		t.Fatal("the slot still holds the used up stack")
	}
	if eq, _ := x.Equippers.Get(holder); len(eq.Equipment()) != 0 {
		t.Fatal("the used up stack is still in hand")
	}
	if err := x.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
	testutil.RemoveScheduled(t, x)
	if x.HasEntity(stack) {
		t.Fatal("the used up stack wasn't removed")
	}
	if want := map[utils.Vec2i]data.TileEnum{{X: 1}: data.Tile_Planks, {X: -1}: data.Tile_Planks}; !maps.Equal(terrain.placed, want) {
		t.Fatalf("placed %v, want %v", terrain.placed, want)
	}
}

func TestAPickedUpPlankDoesntGoIntoAUsedUpStack(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder, stack := newPlankHolder(t, terrain, 1)
	if !place(t, x, holder, utils.Vec2i{X: 1}) {
		t.Fatal("the plank wasn't placed")
	}

	plank, err := x.SpawnItem(data.Item_Planks, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := x.PickUpItem(holder, plank); err != nil || !ok {
		t.Fatalf("picking up a plank: %v, %v", ok, err)
	}
	testutil.RemoveScheduled(t, x)

	inv, _ := x.Inventories.Get(holder)
	if held, _ := inv.Slot(0); held != plank || x.HasEntity(stack) {
		t.Fatalf("slot 0 holds %d, want the plank picked up after the stack ran out", held)
	}
}

func TestPlaceTileParamsAreCopiedNotShared(t *testing.T) {
	p := &abilities.PlaceTileParams{RangeTiles: 3, Tile: data.Tile_Planks}
	c := p.Copy().(*abilities.PlaceTileParams)
	c.RangeTiles = 1
	if p.RangeTiles != 3 {
		t.Fatal("changing a copy changed the original")
	}
}
