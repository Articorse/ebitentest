package systems_test

import (
	"bytes"
	"ebittest/climate"
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"encoding/gob"
	"slices"
	"testing"
)

const suitSlot = data.InventoryColumns

// The player's starting spacesuit, which isn't worn or on the hotbar
func spacesuit(t *testing.T, x *ecs.ECSContainer, player common.EntityId) common.EntityId {
	t.Helper()
	inv, _ := x.Inventories.Get(player)
	suit, ok := inv.Slot(suitSlot)
	if !ok {
		t.Fatal("the player has nothing at the start of the inventory's second row")
	}
	if item, _ := x.Items.Get(suit); x.ItemName(item) != "Spacesuit" {
		t.Fatalf("the start of the inventory's second row holds %q, want the spacesuit", x.ItemName(item))
	}
	return suit
}

func worn(x *ecs.ECSContainer, player common.EntityId, slot ecs.EquipSlotEnum) common.EntityId {
	equipper, _ := x.Equippers.Get(player)
	if e, ok := equipper.InSlot(slot); ok {
		return e
	}
	return -1
}

func TestThePlayerStartsWithASpacesuitThatIsntWorn(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	inv, _ := x.Inventories.Get(player)
	if x.IsWorn(player, suit) || inv.HotbarSlotOf(suit) != -1 {
		t.Fatal("the starting spacesuit is worn or on the hotbar")
	}
	if item, _ := x.Items.Get(suit); x.ItemMass(item) != 10*data.MilligramsPerKg {
		t.Fatalf("the spacesuit weighs %d mg, want 10 kg", x.ItemMass(item))
	}
	if sealed, ok := x.SealedAirs.Get(suit); !ok || sealed.Contents() != (climate.Air{data.Gas_Oxygen: 10 * data.MilligramsPerKg}) {
		t.Fatal("the spacesuit doesn't start out holding 10 kg of oxygen and nothing else")
	}
}

func TestASpacesuitIsWornOnTheTorsoAndStaysInTheInventory(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	carried := x.CarriedMass(player)

	act(t, x, player, ecs.InventoryAction_Wear, suitSlot, int(ecs.Equip_Torso))
	inv, _ := x.Inventories.Get(player)
	if worn(x, player, ecs.Equip_Torso) != suit || !x.IsWorn(player, suit) || slotItem(inv, suitSlot) != suit {
		t.Fatal("the spacesuit isn't worn on the torso while staying in its slot")
	}
	if x.CarriedMass(player) != carried {
		t.Fatalf("wearing the spacesuit changed what's carried from %d to %d mg", carried, x.CarriedMass(player))
	}
	if got := x.WornImages(player); !slices.Equal(got, []common.ImageAssetTag{common.AssetSheetSlimeSpacesuit}) {
		t.Fatalf("what's drawn over the player is %v, want the spacesuit", got)
	}

	act(t, x, player, ecs.InventoryAction_TakeOff, 0, int(ecs.Equip_Torso))
	if worn(x, player, ecs.Equip_Torso) != -1 || slotItem(inv, suitSlot) != suit || len(x.WornImages(player)) != 0 {
		t.Fatal("taking off the spacesuit didn't leave it in its slot, and nothing worn")
	}
}

func TestASpacesuitOnlyFitsTheTorsoOfItsKindOfBody(t *testing.T) {
	for slot := range ecs.EquipSlotCount {
		x, player, _, _, _ := newArmedPlayer(t)
		suit := spacesuit(t, x, player)
		if ok, err := x.Wear(player, suit, slot); err != nil || ok != (slot == ecs.Equip_Torso) {
			t.Errorf("wearing the spacesuit in the %v gave %v (%v)", slot, ok, err)
		}
	}

	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	x.AddComponent(player, ecs.NewBodyComponent(data.Body_Giant))
	if ok, _ := x.Wear(player, suit, ecs.Equip_Torso); ok {
		t.Fatal("a giant wore a spacesuit made for humanoids")
	}
}

func TestAToolOnlyGoesInTheOffHandOfTheSlotsOtherThanTheMainHand(t *testing.T) {
	for slot := range ecs.EquipSlotCount {
		x, player, mattock, _, _ := newArmedPlayer(t)
		if ok, _ := x.Wear(player, mattock, slot); ok != (slot == ecs.Equip_OffHand) {
			t.Errorf("putting the mattock in the %v gave %v", slot, ok)
		}
	}
	x, player, _, _, _ := newArmedPlayer(t)
	if err := x.TakeOff(player, ecs.Equip_MainHand); err == nil {
		t.Fatal("what's in the main hand was taken off rather than following the hotbar")
	}
}

func TestWhatsWornIsntOnTheHotbar(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	inv, _ := x.Inventories.Get(player)
	act(t, x, player, ecs.InventoryAction_Bind, suitSlot, 6)
	if hotbarItem(inv, 6) != suit {
		t.Fatal("the spacesuit couldn't be put on the hotbar before it was worn")
	}

	act(t, x, player, ecs.InventoryAction_Wear, suitSlot, int(ecs.Equip_Torso))
	if hotbarItem(inv, 6) != -1 || !x.IsWorn(player, suit) {
		t.Fatal("wearing the spacesuit didn't take it off the hotbar")
	}
	act(t, x, player, ecs.InventoryAction_Bind, suitSlot, 7)
	if hotbarItem(inv, 7) != -1 {
		t.Fatal("the worn spacesuit was put on the hotbar")
	}
	act(t, x, player, ecs.InventoryAction_Move, suitSlot, 20)
	if worn(x, player, ecs.Equip_Torso) != suit || slotItem(inv, 20) != suit {
		t.Fatal("moving the worn spacesuit to another slot took it off")
	}
}

func TestToggledEquipmentIsWornWhereItFitsOrTakenOff(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	act(t, x, player, ecs.InventoryAction_ToggleWorn, suitSlot, 0)
	if worn(x, player, ecs.Equip_Torso) != suit {
		t.Fatal("toggling the spacesuit didn't wear it")
	}
	act(t, x, player, ecs.InventoryAction_ToggleWorn, suitSlot, 0)
	if worn(x, player, ecs.Equip_Torso) != -1 {
		t.Fatal("toggling the worn spacesuit didn't take it off")
	}
	act(t, x, player, ecs.InventoryAction_ToggleWorn, 0, 0)
	if equipper, _ := x.Equippers.Get(player); len(equipper.Equipment()) != 1 {
		t.Fatal("toggling the mattock wore it")
	}
}

func TestWornEquipmentOnTheHotbarIsInconsistent(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	suit := spacesuit(t, x, player)
	equipper, _ := x.Equippers.Get(player)
	equipper.Equip(ecs.Equip_Torso, suit)
	if err := x.CheckConsistency(); err != nil {
		t.Fatalf("a worn spacesuit is inconsistent: %v", err)
	}
	inv, _ := x.Inventories.Get(player)
	if err := x.BindHotbar(player, 6, suit); err != nil || hotbarItem(inv, 6) != -1 {
		t.Fatalf("binding the worn spacesuit gave %v and bound it", err)
	}
	equipper.Unequip(ecs.Equip_Torso)
	if err := x.BindHotbar(player, 6, suit); err != nil {
		t.Fatal(err)
	}
	equipper.Equip(ecs.Equip_Torso, suit)
	if err := x.CheckConsistency(); err == nil {
		t.Fatal("a worn spacesuit on the hotbar isn't inconsistent")
	}
}

func TestWornEquipmentIsSavedWithWhatItFitsAndHowItsDrawn(t *testing.T) {
	x := testutil.NewECS(t)
	player, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, player); err != nil {
		t.Fatal(err)
	}
	suit := spacesuit(t, x, player)
	eq, _ := x.Equipments.Get(suit)

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(x.GetEntityComponents(suit)); err != nil {
		t.Fatal(err)
	}
	var dtos []ecs.ComponentDto
	if err := gob.NewDecoder(&buf).Decode(&dtos); err != nil {
		t.Fatal(err)
	}
	for _, d := range dtos {
		c, err := ecs.DtoToComponent(d)
		if err != nil {
			t.Fatal(err)
		}
		loaded, ok := c.(*ecs.Equipment)
		if !ok {
			continue
		}
		if loaded.Worn() != eq.Worn() || !loaded.FitsBody(data.Body_Humanoid) || loaded.FitsBody(data.Body_Giant) || !loaded.Fits(ecs.Equip_Torso) {
			t.Fatal("the loaded spacesuit isn't drawn or worn as it was")
		}
		return
	}
	t.Fatal("the spacesuit was saved without its equipment")
}

func equipSlotCenter(slot ecs.EquipSlotEnum) utils.Vec2f {
	cells := map[ecs.EquipSlotEnum]utils.Vec2i{
		ecs.Equip_MainHand: {X: 0, Y: 0}, ecs.Equip_Torso: {X: 1, Y: 0}, ecs.Equip_OffHand: {X: 2, Y: 0},
		ecs.Equip_Ring1: {X: 0, Y: 1}, ecs.Equip_Arms: {X: 1, Y: 1}, ecs.Equip_Ring2: {X: 2, Y: 1},
		ecs.Equip_Legs: {X: 1, Y: 2},
	}
	pad := float64(data.InventoryPanelPaddingPx)
	left := data.HotbarMarginPx + menuSlotPx + data.InventoryMarginPx + pad
	top := menuScreen.Y - data.InventoryMarginPx - pad - float64(3*menuSlotPx+2*data.HotbarSlotGapPx)
	cell := cells[slot]
	return utils.Vec2f{
		X: left + float64(cell.X*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
		Y: top + float64(cell.Y*(menuSlotPx+data.HotbarSlotGapPx)) + menuSlotPx/2,
	}
}

func TestTheEquipmentSlotsSitInTheBottomLeftClearOfTheHotbar(t *testing.T) {
	menu, _, inv, _ := newMenu(t)
	assets := testutil.Assets(t)
	for slot := range ecs.EquipSlotCount {
		at := equipSlotCenter(slot)
		if got, ok := menu.EquipSlotAt(at, menuScreen, assets); !ok || got != slot {
			t.Errorf("at %v is the %v slot (%v), want the %v", at, got, ok, slot)
		}
		if !menu.Covers(at, menuScreen, inv, assets) || menu.SlotAt(at, menuScreen, inv, assets) != -1 {
			t.Errorf("the %v slot isn't the menu's alone", slot)
		}
	}
	lowest := hotbarSlotCenter(data.HotbarSlots - 1)
	if _, ok := menu.EquipSlotAt(lowest, menuScreen, assets); ok {
		t.Fatal("an equipment slot is on the hotbar")
	}
	menu.Toggle()
	if _, ok := menu.EquipSlotAt(equipSlotCenter(ecs.Equip_Torso), menuScreen, assets); ok {
		t.Fatal("the closed menu has equipment slots")
	}
}

func TestCarryingAnItemToAnEquipmentSlotWearsIt(t *testing.T) {
	_, _, _, click := newMenu(t)
	click(inventorySlotCenter(suitSlot))
	if a := click(equipSlotCenter(ecs.Equip_Torso)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Wear, From: suitSlot, To: int(ecs.Equip_Torso)}) {
		t.Fatalf("putting the spacesuit on the torso did %v", a)
	}
	if a := click(equipSlotCenter(ecs.Equip_Legs)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("clicking an equipment slot holding nothing did %v", a)
	}

	click(inventorySlotCenter(suitSlot))
	if a := click(equipSlotCenter(ecs.Equip_MainHand)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("putting the spacesuit in the main hand did %v", a)
	}
	if a := click(inventorySlotCenter(20)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_Move, From: suitSlot, To: 20}) {
		t.Fatalf("after the main hand, putting the spacesuit down did %v, want it still held", a)
	}
}

func TestRightClickingWearsOrTakesOff(t *testing.T) {
	menu, _, inv, click := newMenu(t)
	assets := testutil.Assets(t)
	rightClick := func(at utils.Vec2f) ecs.InventoryAction { return menu.RightClick(at, menuScreen, inv, assets) }

	if a := rightClick(inventorySlotCenter(suitSlot)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_ToggleWorn, From: suitSlot}) {
		t.Fatalf("right clicking the spacesuit did %v", a)
	}
	if a := rightClick(inventorySlotCenter(30)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("right clicking an empty slot did %v", a)
	}
	if a := rightClick(equipSlotCenter(ecs.Equip_Torso)); a != (ecs.InventoryAction{Kind: ecs.InventoryAction_TakeOff, To: int(ecs.Equip_Torso)}) {
		t.Fatalf("right clicking the torso did %v", a)
	}
	if a := rightClick(equipSlotCenter(ecs.Equip_MainHand)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("right clicking the main hand did %v", a)
	}
	click(inventorySlotCenter(suitSlot))
	if a := rightClick(inventorySlotCenter(0)); a.Kind != ecs.InventoryAction_None {
		t.Fatalf("right clicking while holding the spacesuit did %v", a)
	}
}

func TestAnEquipmentSlotsTooltipHasItsNameAndWhatsInIt(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	if got := drawsystem.EquipSlotTooltip(player, ecs.Equip_Torso, x); !slices.Equal(got, []string{"Torso"}) {
		t.Fatalf("the empty torso's tooltip is %v", got)
	}
	act(t, x, player, ecs.InventoryAction_Wear, suitSlot, int(ecs.Equip_Torso))
	if got := drawsystem.EquipSlotTooltip(player, ecs.Equip_Torso, x); !slices.Equal(got, []string{"Torso", "Spacesuit", "Holding 10kg O2, 0mg CO2"}) {
		t.Fatalf("the worn torso's tooltip is %v", got)
	}
	if got := drawsystem.EquipSlotTooltip(player, ecs.Equip_MainHand, x); !slices.Equal(got, []string{"Main hand", "Mattock"}) {
		t.Fatalf("the main hand's tooltip is %v", got)
	}
}

func TestTheOffHandHoldsAToolBesideTheMainHandReadyToUse(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	axe, _ := inv.Slot(3)
	act(t, x, player, ecs.InventoryAction_ToggleWorn, 3, 0)
	if worn(x, player, ecs.Equip_OffHand) != axe || inHand(x, player) != mattock || hotbarItem(inv, 3) != -1 {
		t.Fatal("right clicking the axe didn't put it in the off hand, off the hotbar, beside the mattock in the main hand")
	}
	at := func(e common.EntityId) utils.Vec2f {
		return testutil.WorldPos(t, x, e).Subtract(testutil.WorldPos(t, x, player))
	}
	if got := at(axe); !testutil.ApproxEqual(got, utils.Vec2f{X: data.HeldItemOffsetPx}) {
		t.Fatalf("the axe in the off hand is at %v from the player's middle", got)
	}
	if got := at(mattock); !testutil.ApproxEqual(got, utils.Vec2f{X: -data.HeldItemOffsetPx}) {
		t.Fatalf("the mattock in the main hand is at %v from the player's middle", got)
	}

	if err := x.SelectHotbarSlot(player, 1); err != nil {
		t.Fatal(err)
	}
	if x.IsDisabled(axe) || !x.IsDisabled(mattock) {
		t.Fatal("selecting another hotbar slot put the axe in the off hand away, or left the mattock out")
	}
	target := ecs.AbilityTarget{Pos: testutil.WorldPos(t, x, player).Add(utils.Vec2f{X: 16}), HasPos: true}
	if ok, err := x.ActivateEquipmentAbility(player, ecs.Equip_OffHand, target, 0); err != nil || !ok {
		t.Fatalf("the axe in the off hand couldn't be swung (%v)", err)
	}

	act(t, x, player, ecs.InventoryAction_TakeOff, 0, int(ecs.Equip_OffHand))
	if worn(x, player, ecs.Equip_OffHand) != -1 || !x.IsDisabled(axe) {
		t.Fatal("the axe taken out of the off hand is still held out")
	}
}

func TestAirSealedInWhatsHeldIsntBreathed(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	tank := testutil.AddMovable(t, x, utils.Vec2f{}, false,
		ecs.NewEquipmentComponent(ecs.Equipable_MainHand, [data.MaxEquipmentAbilitySlots]ecs.EntityAbility{}),
		ecs.NewNamedItemComponent("Tank", data.MilligramsPerKg),
		ecs.NewSealedAirComponent(climate.Air{data.Gas_Oxygen: data.MilligramsPerKg}, 1, 2000, data.SpacesuitConductance),
	)
	inv, _ := x.Inventories.Get(player)
	if err := x.PutInInventory(player, inv.FirstFreeSlot(), tank); err != nil {
		t.Fatal(err)
	}
	if err := x.BindHotbar(player, 0, tank); err != nil {
		t.Fatal(err)
	}
	if inHand(x, player) != tank {
		t.Fatal("the tank isn't in hand")
	}
	if _, ok := x.WornSealedAir(player); ok {
		t.Fatal("the air in the tank in hand is breathed")
	}
}
