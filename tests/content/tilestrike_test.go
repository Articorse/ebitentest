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

// Ground everywhere within 5 tiles of the origin, with rock where it's told to be
type fakeTerrain struct {
	rock    map[utils.Vec2i]bool
	damaged map[utils.Vec2i]int
	sturdy  map[utils.Vec2i]bool
}

func newFakeTerrain(rock ...utils.Vec2i) *fakeTerrain {
	f := &fakeTerrain{rock: map[utils.Vec2i]bool{}, damaged: map[utils.Vec2i]int{}, sturdy: map[utils.Vec2i]bool{}}
	for _, r := range rock {
		f.rock[r] = true
	}
	return f
}

func (f *fakeTerrain) loaded(t utils.Vec2i) bool {
	return t.X*t.X+t.Y*t.Y <= 25
}

func (f *fakeTerrain) IsOpaqueTile(t utils.Vec2i) bool {
	return !f.loaded(t) || f.rock[t]
}

func (f *fakeTerrain) TileLayer(t utils.Vec2i) (ecs.LayerMask, bool) {
	if !f.loaded(t) {
		return 0, false
	}
	if f.rock[t] {
		return ecs.Layer_Terrain, true
	}
	return ecs.Layer_Ground, true
}

func (f *fakeTerrain) CanDamageTile(t utils.Vec2i) bool {
	return f.loaded(t) && !f.sturdy[t]
}

func (f *fakeTerrain) DamageTile(t utils.Vec2i, damage int) bool {
	if !f.CanDamageTile(t) {
		return false
	}
	f.damaged[t] += damage
	return true
}

func newMattockHolder(t *testing.T, terrain ecs.Terrain, mask ecs.LayerMask) (*ecs.ECSContainer, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	x.SetTerrain(terrain)

	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{{
		Def:    ecs.NewAbilityDef(abilities.Ability_TileStrike, ecs.Ability_None, 400, 0),
		Params: &abilities.TileStrikeParams{RangeTiles: 2, Damage: 50, Mask: mask},
	}}
	tool := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewEquipmentComponent(ecs.Equipable_MainHand, abis))
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false,
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{ecs.Equip_MainHand: tool}))
	testutil.Attach(t, x, tool, holder, ecs.Link_Held)

	return x, holder
}

func strike(t *testing.T, x *ecs.ECSContainer, holder common.EntityId, tile utils.Vec2i) bool {
	t.Helper()
	used, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{Pos: utils.TilePosToWorldPos(tile), HasPos: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return used
}

func TestTileStrikeHitsTheAimedTileOrTheLastTerrainOnTheWay(t *testing.T) {
	both := ecs.Layer_Terrain | ecs.Layer_Ground
	none := utils.Vec2i{X: 99}
	cases := []struct {
		name   string
		mask   ecs.LayerMask
		rock   []utils.Vec2i
		sturdy []utils.Vec2i
		aim    utils.Vec2i
		want   utils.Vec2i
	}{
		{"its own tile", both, nil, nil, utils.Vec2i{}, utils.Vec2i{}},
		{"ground in range", both, nil, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}},
		{"ground past the range", both, nil, nil, utils.Vec2i{X: 6}, none},
		{"past the range's circle", both, nil, nil, utils.Vec2i{X: 2, Y: 1}, none},
		{"rock past the range", both, []utils.Vec2i{{X: 2}, {X: 4}}, nil, utils.Vec2i{X: 4}, utils.Vec2i{X: 2}},
		{"rock with ground in front", both, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}},
		{"ground behind rock", both, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 1}},
		{"far behind rock", both, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 5}, utils.Vec2i{X: 1}},
		{"ground that can't be damaged", both, nil, []utils.Vec2i{{X: 2}}, utils.Vec2i{X: 2}, none},
		{"rock that can't be damaged", both, []utils.Vec2i{{X: 1}, {X: 2}}, []utils.Vec2i{{X: 2}}, utils.Vec2i{X: 2}, utils.Vec2i{X: 1}},
		{"rock past the aim", ecs.Layer_Terrain, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 1}, none},
		{"rock on the way", ecs.Layer_Terrain, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 4}, utils.Vec2i{X: 2}},
		{"ground behind rock it can't affect", ecs.Layer_Ground, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 3}, none},
	}

	for _, c := range cases {
		terrain := newFakeTerrain(c.rock...)
		for _, s := range c.sturdy {
			terrain.sturdy[s] = true
		}
		x, holder := newMattockHolder(t, terrain, c.mask)

		params := &abilities.TileStrikeParams{RangeTiles: 2, Mask: c.mask}
		tile, ok := params.TargetTile(utils.Vec2f{}, utils.TilePosToWorldPos(c.aim), terrain)
		if !ok {
			tile = none
		}
		if tile != c.want {
			t.Errorf("%s: targets %v, want %v", c.name, tile, c.want)
		}

		want := map[utils.Vec2i]int{c.want: 50}
		if c.want == none {
			want = map[utils.Vec2i]int{}
		}
		if used := strike(t, x, holder, c.aim); used != (c.want != none) || !maps.Equal(terrain.damaged, want) {
			t.Errorf("%s: strike used %v and damaged %v, want %v", c.name, used, terrain.damaged, want)
		}
	}
}

func TestTileStrikeWithoutAnAimDoesNothing(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Ground)

	if used, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{}, 0); err != nil || used || len(terrain.damaged) != 0 {
		t.Fatalf("used %v, error %v, damaged %v; want nothing, though its own tile is at the origin", used, err, terrain.damaged)
	}
}

func TestTileStrikeThatHitsNothingDoesntCoolDown(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Terrain)

	if strike(t, x, holder, utils.Vec2i{X: 2}) {
		t.Fatal("a strike with nothing to hit was used")
	}
	terrain.rock[utils.Vec2i{X: 1}] = true
	if !strike(t, x, holder, utils.Vec2i{X: 2}) {
		t.Fatal("a strike right after one that hit nothing was refused, so that one started the cooldown")
	}
}

func TestTileStrikeCoolsDownAfterBeingUsed(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Ground)

	if !strike(t, x, holder, utils.Vec2i{}) || strike(t, x, holder, utils.Vec2i{}) {
		t.Fatal("want the first strike used and the second refused by the cooldown")
	}
	if want := map[utils.Vec2i]int{{}: 50}; !maps.Equal(terrain.damaged, want) {
		t.Fatalf("damaged %v, want %v", terrain.damaged, want)
	}
}

func TestTileStrikeReachesFromWhoeverHoldsTheTool(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Ground)
	equipper, _ := x.Equippers.Get(holder)
	tool, _ := equipper.InSlot(ecs.Equip_MainHand)
	testutil.SetPos(t, x, holder, utils.TilePosToWorldPos(utils.Vec2i{X: -1}))
	testutil.SetPos(t, x, tool, utils.TilePosToWorldPos(utils.Vec2i{X: 4}))

	strike(t, x, holder, utils.Vec2i{X: -3})
	if want := map[utils.Vec2i]int{{X: -3}: 50}; !maps.Equal(terrain.damaged, want) {
		t.Fatalf("damaged %v, want %v: two tiles from the holder, not from the tool held away from it", terrain.damaged, want)
	}
}

func TestTileStrikeDoesntHurtEntitiesOnTheTile(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Terrain|ecs.Layer_Ground|ecs.Layer_Enemy)
	target := utils.Vec2i{X: 1}
	bystander := testutil.AddMovable(t, x, utils.TilePosToWorldPos(target), false, ecs.NewHitpointsComponent(10, 0))

	if !strike(t, x, holder, target) {
		t.Fatal("strike wasn't used")
	}
	if hp, _ := x.Hitpoints.Get(bystander); hp.Current() != 10 {
		t.Fatalf("the entity on the tile has %d hitpoints, want 10", hp.Current())
	}
}

func TestTileStrikeShowsItsDamageOnlyWhenItLands(t *testing.T) {
	terrain := newFakeTerrain()
	x, holder := newMattockHolder(t, terrain, ecs.Layer_Terrain)

	strike(t, x, holder, utils.Vec2i{X: 2})
	if n := len(x.FloatingTexts.GetEntities()); n != 0 {
		t.Fatalf("a strike that hit nothing spawned %d damage numbers", n)
	}

	terrain.rock[utils.Vec2i{X: 1}] = true
	strike(t, x, holder, utils.Vec2i{X: 2})
	numbers := x.FloatingTexts.GetEntities()
	if len(numbers) != 1 {
		t.Fatalf("a strike spawned %d damage numbers, want 1", len(numbers))
	}
	if pos := testutil.WorldPos(t, x, numbers[0]); pos != utils.TilePosToWorldPos(utils.Vec2i{X: 1}) {
		t.Fatalf("the damage number is at %v, want the struck tile", pos)
	}
}
