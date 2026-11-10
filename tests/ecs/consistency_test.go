package ecs_test

import (
	"ebittest/content/abilities"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"strings"
	"testing"
)

func TestConsistencyCheck(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}
	if err := x.CheckConsistency(); err != nil {
		t.Fatalf("fresh player reported inconsistent: %v", err)
	}

	children := x.Children(p)
	if len(children) != 6 {
		t.Fatalf("player children %v, want the mattock, the bazooka, the gun, the axe, the torches and the spacesuit", children)
	}
	equipment := children[0]
	if err := x.Detach(equipment); err != nil {
		t.Fatal(err)
	}
	testutil.AddMovable(t, x, utils.Vec2f{}, true, ecs.NewTetherComponent(9999))
	testutil.AddMovable(t, x, utils.Vec2f{}, true, ecs.NewTetherComponent(equipment))

	err = x.CheckConsistency()
	if err == nil {
		t.Fatal("inconsistencies not reported")
	}
	for _, want := range []string{"isn't held by it", "tethered to missing entity 9999", "isn't a chunkloader"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("report doesn't mention %q:\n%v", want, err)
		}
	}
}

func TestTheConsistencyCheckFindsImagesTheAssetsDontHave(t *testing.T) {
	sprite := func(tag common.ImageAssetTag, frame int) ecs.Component {
		spr, err := ecs.NewSpriteComponent(tag, 1, false)
		if err != nil {
			t.Fatal(err)
		}
		spr.SetSubImageIdx(frame)
		return spr
	}
	animation := func(frame uint16) ecs.Component {
		ani, err := ecs.NewAnimationComponent(common.AssetSheetSlime, map[ecs.AnimationState][]ecs.AnimationFrame{
			ecs.Anim_Idle: {{FrameIdx: 0, DurationMs: 100}, {FrameIdx: frame, DurationMs: 100}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return ani
	}
	spawner := func(spawned ecs.Component) ecs.Component {
		sp, err := ecs.NewSpawnerComponent(utils.Vec2f{}, ecs.SpawnerType_Point, nil, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), spawned)
		if err != nil {
			t.Fatal(err)
		}
		return sp
	}
	slash := func(tag common.ImageAssetTag) ecs.Component {
		abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{{
			Def:    ecs.NewAbilityDef(abilities.Ability_Slash, ecs.Ability_None, 600, 0),
			Params: &abilities.SlashParams{Image: tag},
		}}
		return ecs.NewEquipmentComponent(ecs.Equipable_MainHand, abis)
	}
	explosion := func(frame uint16) ecs.Component {
		dr, err := ecs.NewDeathrattleComponent(ecs.EntityAbility{
			Def:    ecs.NewAbilityDef(abilities.Ability_Explode, ecs.Ability_None, 0, 0),
			Params: &abilities.ExplodeParams{Animationframes: []ecs.AnimationFrame{{FrameIdx: frame, DurationMs: 50}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return dr
	}

	for _, c := range []struct {
		name    string
		ok, bad ecs.Component
	}{
		{"a sprite", sprite(common.AssetImageTree, 0), sprite("nothing", 0)},
		{"a sprite's frame", sprite(common.AssetSheetSlime, 2), sprite(common.AssetSheetSlime, 3)},
		{"an animation's frame", animation(2), animation(3)},
		{"what a spawner spawns", spawner(sprite(common.AssetSheetRocket, 1)), spawner(sprite("nothing", 0))},
		{"what an ability draws", slash(common.AssetImageSlash), slash("nothing")},
		{"what a deathrattle draws", explosion(6), explosion(7)},
	} {
		x := testutil.NewECS(t)
		testutil.Add(t, x, ecs.NewHierarchyComponent(), c.ok)
		if err := x.CheckConsistency(); err != nil {
			t.Errorf("%s the assets have was refused: %v", c.name, err)
		}
		testutil.Add(t, x, ecs.NewHierarchyComponent(), c.bad)
		if err := x.CheckConsistency(); err == nil {
			t.Errorf("%s the assets don't have was accepted", c.name)
		}

		x.SetAssets(nil)
		if err := x.CheckConsistency(); err != nil {
			t.Errorf("%s was checked without assets to check it against: %v", c.name, err)
		}
	}
}
