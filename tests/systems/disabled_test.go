package systems_test

import (
	"ebittest/content/timerfuncs"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/abilitysystem"
	"ebittest/ecs/systems/commonsystems"
	"ebittest/ecs/systems/damagesystem"
	"ebittest/ecs/systems/movementsystem"
	"ebittest/ecs/systems/timersystem"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// A pair of identical entities, one of them disabled, so that each system's effect shows on one and not the other
func enabledAndDisabled(t *testing.T, x *ecs.ECSContainer, comps func() []ecs.Component) (enabled, disabled common.EntityId) {
	t.Helper()
	enabled = testutil.AddMovable(t, x, utils.Vec2f{}, false, comps()...)
	disabled = testutil.AddMovable(t, x, utils.Vec2f{}, false, comps()...)
	x.SetDisabled(disabled, true)
	return enabled, disabled
}

func TestDisabledEntitiesDontMove(t *testing.T) {
	x := testutil.NewECS(t)
	moving, still := enabledAndDisabled(t, x, func() []ecs.Component { return nil })
	for _, e := range []common.EntityId{moving, still} {
		v, _ := x.Velocities.Get(e)
		v.SetLocalVector(utils.Vec2f{X: 5})
	}

	if err := movementsystem.Tick(x); err != nil {
		t.Fatal(err)
	}

	if testutil.WorldPos(t, x, moving).X == 0 {
		t.Fatal("the enabled entity didn't move")
	}
	if testutil.WorldPos(t, x, still).X != 0 {
		t.Fatal("the disabled entity moved")
	}
}

func TestDisabledTimersDontRun(t *testing.T) {
	x := testutil.NewECS(t)
	ticking, paused := enabledAndDisabled(t, x, func() []ecs.Component {
		timer, err := ecs.NewTimerComponent(data.TickMs, 1, timerfuncs.TimerFunc_Selfdestruct)
		if err != nil {
			t.Fatal(err)
		}
		return []ecs.Component{timer}
	})

	for range 3 {
		if err := timersystem.Tick(x); err != nil {
			t.Fatal(err)
		}
		testutil.RemoveScheduled(t, x)
	}

	if x.HasEntity(ticking) {
		t.Fatal("the enabled entity's timer didn't run out")
	}
	if !x.HasEntity(paused) {
		t.Fatal("the disabled entity's timer ran out")
	}
}

func TestDisabledAbilitiesCantStartButFinishWhatTheyStarted(t *testing.T) {
	x := testutil.NewECS(t)
	const abilityId, postId = ecs.AbilityEnum(1001), ecs.AbilityEnum(1002)
	activated, ended := []common.EntityId{}, []common.EntityId{}
	if err := x.RegisterAbility(abilityId, func(self common.EntityId, _ ecs.AbilityTarget, _ ecs.AbilityParams, _ *ecs.ECSContainer) (bool, error) {
		activated = append(activated, self)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := x.RegisterAbility(postId, func(self common.EntityId, _ ecs.AbilityTarget, _ ecs.AbilityParams, _ *ecs.ECSContainer) (bool, error) {
		ended = append(ended, self)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	abis := [data.MaxAbilitySlots]ecs.EntityAbility{{Def: ecs.NewAbilityDef(abilityId, postId, 2*data.TickMs, 2*data.TickMs)}}
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewAbilitiesComponent(abis))
	if ok, err := x.ActivateAbility(e, ecs.AbilityTarget{}, 0); err != nil || !ok {
		t.Fatalf("activation %v, error %v", ok, err)
	}

	x.SetDisabled(e, true)
	if ok, err := x.ActivateAbility(e, ecs.AbilityTarget{}, 0); err != nil || ok {
		t.Fatalf("a disabled entity's ability activated %v, error %v", ok, err)
	}
	for range 6 {
		if err := abilitysystem.Tick(x); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(ended, []common.EntityId{e}) {
		t.Fatalf("the ability ended for %v while its entity was disabled, want it to finish once", ended)
	}

	x.SetDisabled(e, false)
	if ok, err := x.ActivateAbility(e, ecs.AbilityTarget{}, 0); err != nil || !ok {
		t.Fatalf("the cooldown didn't run out while disabled: activated %v, error %v", ok, err)
	}
	if !slices.Equal(activated, []common.EntityId{e, e}) {
		t.Fatalf("activated %v, want entity %d twice", activated, e)
	}
}

func TestDisabledEntitiesStayOutOfTheSpatialHash(t *testing.T) {
	x := testutil.NewECS(t)
	a, disabled := enabledAndDisabled(t, x, func() []ecs.Component { return nil })
	b := testutil.AddMovable(t, x, utils.Vec2f{}, false)

	grid, err := commonsystems.PopulateSpatialHashGrid(x, data.SpatialHashGridCellSize)
	if err != nil {
		t.Fatal(err)
	}
	pairs, err := commonsystems.GetSHGProximities(grid, x)
	if err != nil {
		t.Fatal(err)
	}

	for _, cell := range grid {
		if slices.Contains(cell, disabled) {
			t.Fatal("the disabled entity is in the spatial hash")
		}
	}
	if !slices.Equal(pairs, []common.EntityPair{{A: a, B: b}}) {
		t.Fatalf("got proximate pairs %v, want only the enabled entities", pairs)
	}
}

func TestDisabledEntitiesNeitherLoseInvulnerabilityNorSee(t *testing.T) {
	x := testutil.NewECS(t)
	x.SetTerrain(openGround{damaged: map[utils.Vec2i]int{}})
	enabled, disabled := enabledAndDisabled(t, x, func() []ecs.Component {
		return []ecs.Component{ecs.NewInvulnerabilityComponent(1000), ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1)}
	})
	for _, e := range []common.EntityId{enabled, disabled} {
		inv, _ := x.Invulnerabilities.Get(e)
		inv.Set(1000)
	}

	damagesystem.Tick(x)
	visionsystem.Tick(x)

	invEnabled, _ := x.Invulnerabilities.Get(enabled)
	invDisabled, _ := x.Invulnerabilities.Get(disabled)
	if invEnabled.RemainingMs() == 1000 || invDisabled.RemainingMs() != 1000 {
		t.Fatalf("invulnerability ticked to %d enabled and %d disabled, want only the enabled one to tick",
			invEnabled.RemainingMs(), invDisabled.RemainingMs())
	}

	vEnabled, _ := x.Visions.Get(enabled)
	vDisabled, _ := x.Visions.Get(disabled)
	if !vEnabled.CanSee(utils.Vec2i{}) || vDisabled.CanSee(utils.Vec2i{}) {
		t.Fatal("vision updated for the disabled entity, or not for the enabled one")
	}
}

func TestDisablingCoversAttachedEntitiesAndIsSaved(t *testing.T) {
	x := testutil.NewECS(t)
	parent := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	child := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	testutil.Attach(t, x, child, parent, ecs.Link_Held)

	x.SetDisabled(parent, true)
	if !x.IsDisabled(child) {
		t.Fatal("disabling the parent left what it holds enabled")
	}

	members, err := x.GetCompositionComponents(parent)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)
	if !y.IsDisabled(parent) || !y.IsDisabled(child) {
		t.Fatal("disabled entities came back enabled after saving")
	}

	x.SetDisabled(parent, false)
	if x.IsDisabled(parent) || x.IsDisabled(child) {
		t.Fatal("enabling the parent didn't enable what it holds")
	}
}

func TestDisabledEquipmentCantBeUsed(t *testing.T) {
	x := testutil.NewECS(t)
	const abilityId = ecs.AbilityEnum(1003)
	used := 0
	if err := x.RegisterAbility(abilityId, func(common.EntityId, ecs.AbilityTarget, ecs.AbilityParams, *ecs.ECSContainer) (bool, error) {
		used++
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	abis := [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{{Def: ecs.NewAbilityDef(abilityId, ecs.Ability_None, 0, 0)}}
	tool := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewEquipmentComponent(ecs.Equipable_MainHand, abis))
	holder := testutil.AddMovable(t, x, utils.Vec2f{}, false,
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{ecs.Equip_MainHand: tool}))
	testutil.Attach(t, x, tool, holder, ecs.Link_Held)

	x.SetDisabled(tool, true)
	if ok, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{}, 0); err != nil || ok || used != 0 {
		t.Fatalf("disabled equipment activated %v, error %v, used %d times", ok, err, used)
	}

	x.SetDisabled(tool, false)
	if ok, err := x.ActivateEquipmentAbility(holder, ecs.Equip_MainHand, ecs.AbilityTarget{}, 0); err != nil || !ok || used != 1 {
		t.Fatalf("enabled equipment activated %v, error %v, used %d times", ok, err, used)
	}
}
