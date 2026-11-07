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

// Ground everywhere within 5 tiles of the origin, with rock and trees where it's told to be
type fakeTerrain struct {
	rock    map[utils.Vec2i]bool
	trees   map[utils.Vec2i]bool
	sturdy  map[utils.Vec2i]bool
	damaged map[utils.Vec2i]int
	// Trees damaged, apart from the ground under them
	treesDamaged map[utils.Vec2i]int
	// Placed tiles block like rock
	placed         map[utils.Vec2i]data.TileEnum
	placedFeatures map[utils.Vec2i]data.FeatureEnum
}

func newFakeTerrain(rock ...utils.Vec2i) *fakeTerrain {
	f := &fakeTerrain{
		rock:           map[utils.Vec2i]bool{},
		trees:          map[utils.Vec2i]bool{},
		sturdy:         map[utils.Vec2i]bool{},
		damaged:        map[utils.Vec2i]int{},
		treesDamaged:   map[utils.Vec2i]int{},
		placed:         map[utils.Vec2i]data.TileEnum{},
		placedFeatures: map[utils.Vec2i]data.FeatureEnum{},
	}
	for _, r := range rock {
		f.rock[r] = true
	}
	return f
}

func (f *fakeTerrain) loaded(t utils.Vec2i) bool {
	return t.X*t.X+t.Y*t.Y <= 25
}

func (f *fakeTerrain) IsOpenToSky(t utils.Vec2i) bool {
	return f.loaded(t)
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

func (f *fakeTerrain) IsBlockedAt(pos utils.Vec2f) bool {
	t := utils.WorldPosToTilePos(pos)
	return !f.loaded(t) || f.rock[t]
}

func (f *fakeTerrain) CanDamageFeature(t utils.Vec2i) bool {
	return f.trees[t]
}

func (f *fakeTerrain) DamageFeature(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	if !f.trees[t] {
		return false, nil
	}
	f.treesDamaged[t] += damage
	return true, nil
}

func (f *fakeTerrain) CanDamageTile(t utils.Vec2i) bool {
	return f.loaded(t) && !f.sturdy[t]
}

func (f *fakeTerrain) DamageTile(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	if !f.CanDamageTile(t) {
		return false, nil
	}
	f.damaged[t] += damage
	return true, nil
}

func (f *fakeTerrain) CanPlaceTile(t utils.Vec2i) bool {
	return f.loaded(t) && !f.rock[t] && !f.trees[t]
}

func (f *fakeTerrain) PlaceTile(t utils.Vec2i, tile data.TileEnum) bool {
	if !f.CanPlaceTile(t) {
		return false
	}
	f.rock[t] = true
	f.placed[t] = tile
	return true
}

func (f *fakeTerrain) CanPlaceFeature(t utils.Vec2i) bool {
	_, placed := f.placedFeatures[t]
	return f.CanPlaceTile(t) && !placed
}

func (f *fakeTerrain) PlaceFeature(t utils.Vec2i, feature data.FeatureEnum) bool {
	if !f.CanPlaceFeature(t) {
		return false
	}
	f.placedFeatures[t] = feature
	return true
}

func (f *fakeTerrain) LightSources(utils.Vec2i, utils.Vec2i, func(utils.Vec2i, uint8, int)) {}
func (f *fakeTerrain) CanPlaceSite(utils.Vec2i) bool                                        { return false }
func (f *fakeTerrain) PlaceSite(utils.Vec2i, data.BlueprintEnum) bool                       { return false }
func (f *fakeTerrain) Site(utils.Vec2i) (ecs.ConstructionSite, bool) {
	return ecs.ConstructionSite{}, false
}
func (f *fakeTerrain) SitesIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i        { return nil }
func (f *fakeTerrain) InArea(data.AreaEnum, utils.Vec2i) bool                { return false }
func (f *fakeTerrain) SetArea(data.AreaEnum, utils.Vec2i, utils.Vec2i, bool) {}
func (f *fakeTerrain) FeaturesIn(utils.Vec2i, utils.Vec2i, data.FeatureEnum) []utils.Vec2i {
	return nil
}
func (f *fakeTerrain) GroundStack(utils.Vec2i) (ecs.GroundStack, bool) {
	return ecs.GroundStack{}, false
}
func (f *fakeTerrain) GroundStacksIn(utils.Vec2i, utils.Vec2i) []utils.Vec2i {
	return nil
}
func (f *fakeTerrain) SetGroundStack(utils.Vec2i, ecs.GroundStack) bool {
	return false
}
func (f *fakeTerrain) AreaTiles(data.AreaEnum, utils.Vec2i, utils.Vec2i) []utils.Vec2i {
	return nil
}
func (f *fakeTerrain) RemoveSite(utils.Vec2i) bool                           { return false }
func (f *fakeTerrain) DeliverToSite(utils.Vec2i, data.ItemEnum, int) int     { return 0 }
func (f *fakeTerrain) BuildSite(utils.Vec2i, int, bool) (worked, built bool) { return false, false }

func (f *fakeTerrain) WalkSpeedPercent(t utils.Vec2i) int {
	if !f.loaded(t) || f.rock[t] || f.trees[t] {
		return 0
	}
	return 100
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
	all := ecs.Layer_Terrain | ecs.Layer_Ground | ecs.Layer_Feature
	none := utils.Vec2i{X: 99}
	cases := []struct {
		name   string
		mask   ecs.LayerMask
		rock   []utils.Vec2i
		sturdy []utils.Vec2i
		aim    utils.Vec2i
		want   utils.Vec2i
		trees  []utils.Vec2i
	}{
		{"its own tile", all, nil, nil, utils.Vec2i{}, utils.Vec2i{}, nil},
		{"ground in range", all, nil, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}, nil},
		{"ground past the range", all, nil, nil, utils.Vec2i{X: 6}, none, nil},
		{"past the range's circle", all, nil, nil, utils.Vec2i{X: 2, Y: 1}, none, nil},
		{"rock past the range", all, []utils.Vec2i{{X: 2}, {X: 4}}, nil, utils.Vec2i{X: 4}, utils.Vec2i{X: 2}, nil},
		{"rock with ground in front", all, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}, nil},
		{"ground behind rock", all, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 1}, nil},
		{"far behind rock", all, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 5}, utils.Vec2i{X: 1}, nil},
		{"ground that can't be damaged", all, nil, []utils.Vec2i{{X: 2}}, utils.Vec2i{X: 2}, none, nil},
		{"rock that can't be damaged", all, []utils.Vec2i{{X: 1}, {X: 2}}, []utils.Vec2i{{X: 2}}, utils.Vec2i{X: 2}, utils.Vec2i{X: 1}, nil},
		{"rock past the aim", ecs.Layer_Terrain, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 1}, none, nil},
		{"rock on the way", ecs.Layer_Terrain, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 4}, utils.Vec2i{X: 2}, nil},
		{"ground behind rock it can't affect", ecs.Layer_Ground, []utils.Vec2i{{X: 1}}, nil, utils.Vec2i{X: 3}, none, nil},
		{"a tree", all, nil, nil, utils.Vec2i{X: 1}, utils.Vec2i{X: 1}, []utils.Vec2i{{X: 1}}},
		{"rock behind a tree", all, []utils.Vec2i{{X: 2}}, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}, []utils.Vec2i{{X: 1}}},
		{"ground behind a tree", all, nil, nil, utils.Vec2i{X: 2}, utils.Vec2i{X: 2}, []utils.Vec2i{{X: 1}}},
		{"a tree past the range", all, nil, nil, utils.Vec2i{X: 5}, utils.Vec2i{X: 2}, []utils.Vec2i{{X: 2}}},
		{"a tree with a ground-only mask", ecs.Layer_Ground, nil, nil, utils.Vec2i{X: 1}, none, []utils.Vec2i{{X: 1}}},
		{"a tree with a mask without features", ecs.Layer_Terrain | ecs.Layer_Ground, nil, nil, utils.Vec2i{X: 1}, none, []utils.Vec2i{{X: 1}}},
		{"a tree on ground that can't be damaged", all, nil, []utils.Vec2i{{X: 1}}, utils.Vec2i{X: 1}, utils.Vec2i{X: 1}, []utils.Vec2i{{X: 1}}},
	}

	for _, c := range cases {
		terrain := newFakeTerrain(c.rock...)
		for _, s := range c.sturdy {
			terrain.sturdy[s] = true
		}
		for _, tree := range c.trees {
			terrain.trees[tree] = true
		}
		x, holder := newMattockHolder(t, terrain, c.mask)

		params := &abilities.TileStrikeParams{RangeTiles: 2, Mask: c.mask}
		tile, ok := params.TargetTile(utils.Vec2f{}, utils.TilePosToWorldPos(c.aim), x)
		if !ok {
			tile = none
		}
		if tile != c.want {
			t.Errorf("%s: targets %v, want %v", c.name, tile, c.want)
		}

		wantGround, wantTrees := map[utils.Vec2i]int{}, map[utils.Vec2i]int{}
		switch {
		case c.want == none:
		case terrain.trees[c.want]:
			wantTrees[c.want] = 50
		default:
			wantGround[c.want] = 50
		}
		used := strike(t, x, holder, c.aim)
		if used != (c.want != none) || !maps.Equal(terrain.damaged, wantGround) || !maps.Equal(terrain.treesDamaged, wantTrees) {
			t.Errorf("%s: strike used %v, damaged ground %v and trees %v; want ground %v and trees %v",
				c.name, used, terrain.damaged, terrain.treesDamaged, wantGround, wantTrees)
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
	bystander := testutil.AddMovable(t, x, utils.TilePosToWorldPos(target), false, testutil.DummyBody(t, x, 10))

	if !strike(t, x, holder, target) {
		t.Fatal("strike wasn't used")
	}
	if health := testutil.PartHealth(t, x, bystander, testutil.DummyPart); health != 10 {
		t.Fatalf("the entity on the tile has %d health, want 10", health)
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

func TestTheStartingMattockStrikesTrees(t *testing.T) {
	x, player, _ := newAxeHolder(t)
	terrain := newFakeTerrain()
	terrain.trees[utils.Vec2i{X: 1}] = true
	x.SetTerrain(terrain)
	if err := x.SelectHotbarSlot(player, 0); err != nil {
		t.Fatal(err)
	}

	targeting, ok := x.HeldTileTargeting(player)
	if !ok {
		t.Fatal("the mattock in hand doesn't target tiles")
	}
	if tile, ok := targeting.TargetTile(utils.Vec2f{}, utils.TilePosToWorldPos(utils.Vec2i{X: 1}), x); !ok || tile != (utils.Vec2i{X: 1}) {
		t.Fatalf("aiming at a tree, the mattock targets %v (%v), want the tree", tile, ok)
	}
}
