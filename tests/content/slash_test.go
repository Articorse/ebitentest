package content_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/collisionsystem"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"slices"
	"testing"
)

// A player at the origin with the axe from its starting equipment in hand
func newAxeHolder(t *testing.T) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x := testutil.NewECS(t)
	player, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	testutil.SetPos(t, x, player, utils.Vec2f{})
	if err := templates.SpawnStartingEquipment(x, player); err != nil {
		t.Fatal(err)
	}
	if err := x.SelectInventorySlot(player, 3); err != nil {
		t.Fatal(err)
	}

	inv, _ := x.Inventories.Get(player)
	axe, _ := inv.Slot(3)
	return x, player, axe
}

// Returns the slash, or -1 if the axe wasn't used
func slash(t *testing.T, x *ecs.ECSContainer, holder common.EntityId, target ecs.AbilityTarget) common.EntityId {
	t.Helper()
	before := x.HurtboxColliders.GetEntities()
	used, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, target, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !used {
		return -1
	}

	for _, e := range x.HurtboxColliders.GetEntities() {
		if !slices.Contains(before, e) {
			return e
		}
	}
	t.Fatal("the axe was used, but left no slash")
	return -1
}

func aimAt(pos utils.Vec2f) ecs.AbilityTarget {
	return ecs.AbilityTarget{Pos: pos, HasPos: true}
}

func addEnemy(t *testing.T, x *ecs.ECSContainer, pos utils.Vec2f) common.EntityId {
	t.Helper()
	shape, err := shapes.NewCircleShape(5, utils.Vec2f{})
	if err != nil {
		t.Fatal(err)
	}
	return testutil.AddMovable(t, x, pos, false, ecs.NewHitpointsComponent(100, 0),
		ecs.NewHitboxColliderComponent(ecs.Layer_Enemy, ecs.Layer_FriendlyProjectile, shape))
}

// One tick of the slash against the enemies, through the same collision checks as the game
func cut(t *testing.T, x *ecs.ECSContainer, slash common.EntityId, enemies ...common.EntityId) {
	t.Helper()
	pairs := []common.EntityPair{}
	for _, e := range enemies {
		pairs = append(pairs, common.EntityPair{A: slash, B: e})
	}
	hurtboxes, hitboxes := ecs.Colliders(&x.HurtboxColliders), ecs.Colliders(&x.HitboxColliders)
	aabb := collisionsystem.GetDirectedAABBCollisions(hurtboxes, hitboxes, pairs, x)
	if _, err := damagesystem.DealContactDamage(collisionsystem.GetCollisions(hurtboxes, hitboxes, aabb, x), nil, x); err != nil {
		t.Fatal(err)
	}
}

func hurt(x *ecs.ECSContainer, e common.EntityId) int {
	hp, _ := x.Hitpoints.Get(e)
	return 100 - hp.Current()
}

func TestTheAxeSlashesTowardTheAimFromWhoeverHoldsIt(t *testing.T) {
	for _, aim := range []utils.Vec2f{{X: 50}, {Y: -50}, {X: -30, Y: 30}} {
		x, player, axe := newAxeHolder(t)
		holderPos := utils.Vec2f{X: 40, Y: 8}
		testutil.SetPos(t, x, player, holderPos)
		testutil.SetPos(t, x, axe, utils.Vec2f{X: 30})

		s := slash(t, x, player, aimAt(holderPos.Add(aim)))
		if s == -1 {
			t.Fatalf("aiming %v: the axe wasn't used", aim)
		}

		dir := aim.Normalized()
		if pos := testutil.WorldPos(t, x, s); !testutil.ApproxEqual(pos, holderPos.Add(dir.Multiply(14))) {
			t.Errorf("aiming %v: the slash is at %v, want 14px from the holder toward the aim", aim, pos)
		}
		tr, _ := x.Transforms.Get(s)
		if rot := tr.LocalRotation(); math.Abs(rot-math.Atan2(dir.Y, dir.X)) > 1e-9 {
			t.Errorf("aiming %v: the slash is turned %v, want toward the aim", aim, rot)
		}
		spr, ok := x.Sprites.Get(s)
		axeSpr, _ := x.Sprites.Get(axe)
		if !ok || spr.ImageAsset() != common.AssetImageSlash || !spr.AllowRotation() || spr.Layer() != axeSpr.Layer()+1 {
			t.Errorf("aiming %v: the slash isn't drawn turned, as a slash, over the axe", aim)
		}

		ahead := addEnemy(t, x, holderPos.Add(dir.Multiply(20)))
		behind := addEnemy(t, x, holderPos.Subtract(dir.Multiply(20)))
		beyond := addEnemy(t, x, holderPos.Add(dir.Multiply(32)))
		cut(t, x, s, ahead, behind, beyond)
		if hurt(x, ahead) != 50 || hurt(x, behind) != 0 || hurt(x, beyond) != 0 {
			t.Errorf("aiming %v: the slash hurt the enemy ahead %d, behind %d and beyond its reach %d; want 50, 0, 0",
				aim, hurt(x, ahead), hurt(x, behind), hurt(x, beyond))
		}
	}
}

func TestTheSlashCoversTheCrescentsTipsButNotPastThem(t *testing.T) {
	x, player, _ := newAxeHolder(t)
	s := slash(t, x, player, aimAt(utils.Vec2f{X: 50}))

	// The tips of the crescent reach 18px to each side of the aim, 4px out from the holder
	tips := []common.EntityId{addEnemy(t, x, utils.Vec2f{X: 6, Y: 20}), addEnemy(t, x, utils.Vec2f{X: 6, Y: -20})}
	past := []common.EntityId{addEnemy(t, x, utils.Vec2f{X: 6, Y: 25}), addEnemy(t, x, utils.Vec2f{X: 6, Y: -25})}
	cut(t, x, s, append(tips, past...)...)

	for _, e := range tips {
		if hurt(x, e) != 50 {
			t.Errorf("an enemy at a tip of the slash lost %d, want 50", hurt(x, e))
		}
	}
	for _, e := range past {
		if hurt(x, e) != 0 {
			t.Errorf("an enemy past a tip of the slash lost %d, want 0", hurt(x, e))
		}
	}
}

func TestTheSlashOnlyHurtsOnTheTickItAppearsAndFadesAfter150ms(t *testing.T) {
	x, player, _ := newAxeHolder(t)
	s := slash(t, x, player, aimAt(utils.Vec2f{X: 50}))
	cut(t, x, s)

	late := addEnemy(t, x, utils.Vec2f{X: 20})
	cut(t, x, s, late)
	if hurt(x, late) != 0 {
		t.Fatalf("an enemy that came into the slash after its first tick lost %d", hurt(x, late))
	}

	tick := func(n int) {
		t.Helper()
		for range n {
			if err := timersystem.Tick(x); err != nil {
				t.Fatal(err)
			}
			testutil.RemoveScheduled(t, x)
		}
	}
	tick(150/data.TickMs - 1)
	if !x.HasEntity(s) {
		t.Fatal("the slash faded before 150ms")
	}
	tick(2)
	if x.HasEntity(s) {
		t.Fatal("the slash is still there after 150ms")
	}
}

func TestTheAxeNeedsAnAimAndCoolsDownFor600ms(t *testing.T) {
	x, player, axe := newAxeHolder(t)
	if slash(t, x, player, ecs.AbilityTarget{}) != -1 || slash(t, x, player, aimAt(utils.Vec2f{})) != -1 {
		t.Fatal("the axe slashed without an aim to slash toward")
	}
	if !x.Swings.HasComponent(axe) {
		t.Fatal("the axe doesn't swing")
	}
	if _, ok := x.HeldTileTargeting(player); ok {
		t.Fatal("the axe in hand targets tiles")
	}

	if slash(t, x, player, aimAt(utils.Vec2f{X: 50})) == -1 {
		t.Fatal("the axe wasn't used")
	}
	tick := func(ms int) {
		t.Helper()
		for range ms / data.TickMs {
			if err := abilitysystem.Tick(x); err != nil {
				t.Fatal(err)
			}
		}
	}
	tick(592)
	if slash(t, x, player, aimAt(utils.Vec2f{X: 50})) != -1 {
		t.Fatal("the axe was used again before its 600ms cooldown ran out")
	}
	tick(32)
	if slash(t, x, player, aimAt(utils.Vec2f{X: 50})) == -1 {
		t.Fatal("the axe couldn't be used after its cooldown")
	}
}
