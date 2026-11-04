package content_test

import (
	"ebittest/content/abilities"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"maps"
	"testing"
)

// A holder at the origin with a stack of torches in its first slot, in hand
func newTorchHolder(t *testing.T, terrain ecs.Terrain, torches int) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	x.SetTerrain(terrain)

	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false,
		ecs.NewInventoryComponent(2, 2),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}))
	stack, err := x.SpawnItemStack(data.Item_Torch, torches, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := x.PickUpItem(holder, stack); err != nil || !ok {
		t.Fatalf("picking up torches: %v, %v", ok, err)
	}
	return x, holder, stack
}

func TestTorchesArePlacedOnFreeGroundInReachAndSightEvenUnderSomeone(t *testing.T) {
	cases := []struct {
		name     string
		rock     []utils.Vec2i
		trees    []utils.Vec2i
		torches  []utils.Vec2i
		blockers []utils.Vec2f
		items    []utils.Vec2f
		aim      utils.Vec2i
		placed   bool
	}{
		{"empty ground", nil, nil, nil, nil, nil, utils.Vec2i{X: 2}, true},
		{"empty ground at the edge of reach", nil, nil, nil, nil, nil, utils.Vec2i{X: 3}, true},
		{"the holder's own tile", nil, nil, nil, nil, nil, utils.Vec2i{}, true},
		{"ground out of reach", nil, nil, nil, nil, nil, utils.Vec2i{X: 4}, false},
		{"rock", []utils.Vec2i{{X: 2}}, nil, nil, nil, nil, utils.Vec2i{X: 2}, false},
		{"ground behind rock", []utils.Vec2i{{X: 1}}, nil, nil, nil, nil, utils.Vec2i{X: 2}, false},
		{"a tree", nil, []utils.Vec2i{{X: 2}}, nil, nil, nil, utils.Vec2i{X: 2}, false},
		{"another torch", nil, nil, []utils.Vec2i{{X: 2}}, nil, nil, utils.Vec2i{X: 2}, false},
		{"ground with something standing on it", nil, nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 2})}, nil, utils.Vec2i{X: 2}, true},
		{"ground with an item on it", nil, nil, nil, nil, []utils.Vec2f{utils.TilePosToWorldPos(utils.Vec2i{X: 2})}, utils.Vec2i{X: 2}, true},
	}

	for _, c := range cases {
		terrain := newFakeTerrain(c.rock...)
		for _, tree := range c.trees {
			terrain.trees[tree] = true
		}
		for _, torch := range c.torches {
			terrain.placedFeatures[torch] = data.Feature_Torch
		}
		x, holder, stack := newTorchHolder(t, terrain, 2)
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
			t.Fatal("torches in hand don't target tiles")
		}
		tile, targeted := targeting.TargetTile(utils.Vec2f{}, utils.TilePosToWorldPos(c.aim), x)
		if targeted != c.placed || (targeted && tile != c.aim) {
			t.Errorf("%s: targets %v, %v; want %v", c.name, tile, targeted, c.placed)
		}

		want := maps.Clone(terrain.placedFeatures)
		left := 2
		if c.placed {
			want[c.aim] = data.Feature_Torch
			left = 1
		}
		used := place(t, x, holder, c.aim)
		if item, _ := x.Items.Get(stack); used != c.placed || !maps.Equal(terrain.placedFeatures, want) || item.Count() != left {
			t.Errorf("%s: placing used %v, placed %v and left %d torches; want %v and %d", c.name, used, terrain.placedFeatures, item.Count(), want, left)
		}
		if len(terrain.placed) != 0 {
			t.Errorf("%s: a torch placed tiles %v", c.name, terrain.placed)
		}
	}
}

func TestPlaceFeatureParamsAreCopiedNotShared(t *testing.T) {
	p := &abilities.PlaceFeatureParams{RangeTiles: 3, Feature: data.Feature_Torch}
	c := p.Copy().(*abilities.PlaceFeatureParams)
	c.RangeTiles = 1
	if p.RangeTiles != 3 || p.PlacedFeature() != data.Feature_Torch {
		t.Fatal("changing the copy changed the original")
	}
}
