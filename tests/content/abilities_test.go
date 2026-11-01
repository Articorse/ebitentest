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
	"testing"
)

func TestDodgeIsInvulnerableOnlyWhileActive(t *testing.T) {
	x := testutil.NewECS(t)
	spr, err := ecs.NewSpriteComponent(common.AssetSheetSlime, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	frames := map[ecs.AnimationState][]ecs.AnimationFrame{
		ecs.Anim_Idle: {{FrameIdx: 0, DurationMs: 100}},
		ecs.Anim_Jump: {{FrameIdx: 0, DurationMs: 50}},
	}
	ani, err := ecs.NewAnimationComponent(common.AssetSheetSlime, frames)
	if err != nil {
		t.Fatal(err)
	}
	const durationMs = 200
	abis := [data.MaxAbilitySlots]ecs.EntityAbility{{
		Def:    ecs.NewAbilityDef(abilities.Ability_Dodge, abilities.Ability_Dodge_Post, 1000, durationMs),
		Params: &abilities.DodgeParams{Force: 10},
	}}
	p := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewDefaultVelocityComponent(), spr, ani,
		ecs.NewHitpointsComponent(20, 1000), ecs.NewAbilitiesComponent(abis))

	x.SetTickInputs(0, map[common.EntityId]ecs.InputState{p: {Analog1X: 1}})
	if activated, err := x.AbilitiesManager.ActivateAbility(p, nil, utils.Vec2f{}, 0, x); err != nil || !activated {
		t.Fatalf("dodge activated %v, error %v", activated, err)
	}

	invulAt := func() int {
		t.Helper()
		invul, err := x.HitpointsManager.GetInvulCurrent(p, x)
		if err != nil {
			t.Fatal(err)
		}
		return invul
	}

	if invulAt() <= 0 {
		t.Fatal("not invulnerable right after dodging")
	}

	for range durationMs/data.TickMs + 1 {
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
	}

	if invul := invulAt(); invul != 0 {
		t.Fatalf("still invulnerable for %d ms after the dodge ended", invul)
	}
}

func TestSpawnAbilitySpawnsOncePerActivation(t *testing.T) {
	x := testutil.NewECS(t)
	sp, err := ecs.NewSpawnerComponent(utils.Vec2f{}, ecs.SpawnerType_Point, nil, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{{
		Def: ecs.NewAbilityDef(abilities.Ability_Spawn, ecs.Ability_None, 500, 0),
	}}
	gun := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), sp, ecs.NewEquipmentComponent(ecs.Equipable_MainHand, abis))
	holder := testutil.Add(t, x, ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{ecs.Equip_MainHand: gun}))

	before := len(x.Transforms.GetEntities())
	if _, err := x.EquipManager.ActivateAbility(holder, ecs.Equip_MainHand, nil, utils.Vec2f{}, 0, x); err != nil {
		t.Fatal(err)
	}
	if err := abilitysystem.Tick(x); err != nil {
		t.Fatal(err)
	}

	if spawned := len(x.Transforms.GetEntities()) - before; spawned != 1 {
		t.Fatalf("spawned %d entities, want 1", spawned)
	}
}

func newRocket(t *testing.T, x *ecs.ECSContainer) common.EntityId {
	t.Helper()

	hurtShape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	dr, err := ecs.NewDeathrattleComponent(ecs.EntityAbility{
		Def: ecs.NewAbilityDef(abilities.Ability_Explode, ecs.Ability_None, 0, 0),
		Params: &abilities.ExplodeParams{
			Radii:           []float64{10},
			DmgTiers:        []int{1},
			Animationframes: []ecs.AnimationFrame{{FrameIdx: 0, DurationMs: 50}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0),
		ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Enemy, hurtShape), dr)
}

func TestDeathrattleRunsOnKillOnly(t *testing.T) {
	cases := []struct {
		name           string
		remove, kill   bool
		heldRocket     bool
		wantExplosions int
	}{
		{name: "removed", remove: true, wantExplosions: 0},
		{name: "killed", kill: true, wantExplosions: 1},
		{name: "removed and killed in the same tick", remove: true, kill: true, wantExplosions: 1},
		{name: "killed while holding another rocket, which is dropped", kill: true, heldRocket: true, wantExplosions: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := testutil.NewECS(t)
			r := newRocket(t, x)
			if c.heldRocket {
				testutil.Attach(t, x, newRocket(t, x), r, ecs.Link_Held)
			}

			if c.remove {
				x.ScheduleRemoveEntity(r)
			}
			if c.kill {
				x.ScheduleKillEntity(r)
			}
			testutil.RemoveScheduled(t, x)

			if explosions := len(x.Timers.GetEntities()); explosions != c.wantExplosions {
				t.Fatalf("%d explosions, want %d", explosions, c.wantExplosions)
			}
		})
	}
}
