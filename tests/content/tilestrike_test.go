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

func (f *fakeTerrain) DamageTile(t utils.Vec2i, damage int) bool {
	if !f.loaded(t) || f.sturdy[t] {
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
	used, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{Pos: utils.TilePosToWorldPos(tile)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return used
}

func TestTileStrikeDamagesTilesItCanReachAndSee(t *testing.T) {
	for _, tile := range []utils.Vec2i{{}, {X: 2}, {X: -1, Y: 1}, {Y: -2}, {X: 1, Y: 0}} {
		terrain := newFakeTerrain(utils.Vec2i{X: 1, Y: 0})
		x, holder := newMattockHolder(t, terrain, ecs.Layer_Terrain|ecs.Layer_Ground)
		if tile == (utils.Vec2i{X: 2}) {
			delete(terrain.rock, utils.Vec2i{X: 1, Y: 0})
		}

		if !strike(t, x, holder, tile) {
			t.Errorf("striking %v wasn't used", tile)
			continue
		}
		if want := map[utils.Vec2i]int{tile: 50}; !maps.Equal(terrain.damaged, want) {
			t.Errorf("striking %v damaged %v, want %v", tile, terrain.damaged, want)
		}
	}
}

func TestTileStrikeAtSomethingItCantAffectDoesNothingAndDoesntCoolDown(t *testing.T) {
	behindRock, rock := utils.Vec2i{X: 2}, utils.Vec2i{X: 1}
	cases := []struct {
		name string
		tile utils.Vec2i
		mask ecs.LayerMask
		edit func(f *fakeTerrain)
	}{
		{"out of range", utils.Vec2i{X: 3}, ecs.Layer_Terrain | ecs.Layer_Ground, nil},
		{"out of the range's circle", utils.Vec2i{X: 2, Y: 1}, ecs.Layer_Terrain | ecs.Layer_Ground, nil},
		{"ground behind rock", behindRock, ecs.Layer_Terrain | ecs.Layer_Ground, func(f *fakeTerrain) { f.rock[rock] = true }},
		{"rock behind rock", behindRock, ecs.Layer_Terrain | ecs.Layer_Ground, func(f *fakeTerrain) { f.rock[rock], f.rock[behindRock] = true, true }},
		{"ground with a terrain mask", utils.Vec2i{X: 1}, ecs.Layer_Terrain, nil},
		{"rock with a ground mask", rock, ecs.Layer_Ground, func(f *fakeTerrain) { f.rock[rock] = true }},
		{"a tile that can't be damaged", utils.Vec2i{X: 1}, ecs.Layer_Terrain | ecs.Layer_Ground, func(f *fakeTerrain) { f.sturdy[utils.Vec2i{X: 1}] = true }},
	}

	for _, c := range cases {
		reachableRock := utils.Vec2i{Y: -1}
		terrain := newFakeTerrain(reachableRock)
		if c.edit != nil {
			c.edit(terrain)
		}
		x, holder := newMattockHolder(t, terrain, c.mask)

		if strike(t, x, holder, c.tile) || len(terrain.damaged) != 0 {
			t.Errorf("%s: strike was used, damaged %v", c.name, terrain.damaged)
			continue
		}
		valid := utils.Vec2i{}
		if c.mask&ecs.Layer_Ground == 0 {
			valid = reachableRock
		}
		if !strike(t, x, holder, valid) {
			t.Errorf("%s: a valid strike right after was refused, so the refused one started the cooldown", c.name)
		}
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
	testutil.SetPos(t, x, tool, utils.TilePosToWorldPos(utils.Vec2i{X: 3}))

	if strike(t, x, holder, utils.Vec2i{X: 3}) || !strike(t, x, holder, utils.Vec2i{X: 1}) {
		t.Fatal("want the strike to reach two tiles from the holder, not from the tool held away from it")
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
