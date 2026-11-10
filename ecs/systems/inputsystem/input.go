package inputsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/facingsystem"
	"ebittest/utils"
	"fmt"
	"log"
	"maps"
	"math"
	"slices"
)

type equipTrigger struct {
	value  float64
	slot   ecs.EquipSlotEnum
	abiIdx int
	label  string
}

type selfTrigger struct {
	value  float64
	abiIdx int
	label  string
}

// Facing is updated between taking aim and using abilities, so that what's used this tick points where it turned to
// this tick
func HandleInputs(
	camera utils.Vec2f,
	ecsContainer *ecs.ECSContainer,
	allInputs map[common.EntityId]ecs.InputState,
) error {
	// In entity order, so that abilities activate, and spawned entities get their IDs, the same way every run
	entities := slices.Sorted(maps.Keys(allInputs))

	for _, e := range entities {
		input := allInputs[e]

		// Before facing and using equipment, so that a newly selected item is aimed before its first use
		if inv, ok := ecsContainer.Inventories.Get(e); ok {
			slot := inv.Selected()
			if input.SelectSlot > 0 {
				slot = input.SelectSlot - 1
			} else if size := inv.HotbarSize(); size > 0 {
				slot = ((slot+input.ScrollSlot)%size + size) % size
			}
			if slot != inv.Selected() {
				if err := ecsContainer.SelectHotbarSlot(e, slot); err != nil {
					log.Printf("error selecting slot %d of entity %d: %v\n", slot+1, e, err)
				}
			}
		}

		if aim, ok := ecsContainer.Aims.Get(e); ok {
			if input.HasAim {
				aim.SetTarget(input.AimPos)
			} else {
				aim.ClearTarget()
			}
		}

		if p := input.PlaceBlueprint; p.Kind != data.Blueprint_None {
			ecsContainer.LastOrder++
			seq := 0
			for _, t := range p.Tiles {
				if ecsContainer.Terrain().PlaceSite(t, p.Kind) {
					ecsContainer.Terrain().SetSiteOrder(t, ecs.SiteOrder{Id: ecsContainer.LastOrder, Seq: seq})
					if f, ok := ecsContainer.Factions.Get(e); ok {
						ecsContainer.Terrain().SetSiteFaction(t, f.Faction())
					}
					seq++
				}
			}
		}
		if r := input.RemoveBlueprint; r.Remove {
			ecsContainer.Terrain().RemoveSite(r.Tile)
		}
		if a := input.Area; a.Area != data.Area_None {
			ecsContainer.MarkArea(a.Area, a.First, a.Last, !a.Erase)
		}

		if m := input.Storage; m.Mark {
			for _, t := range m.Tiles {
				if m.Erase || ecsContainer.CanPlaceStorage(t) {
					ecsContainer.MarkArea(data.Area_Storage, t, t, !m.Erase)
				}
			}
		}

		if p := input.PickUp; p.PickUp {
			if err := pickUpInReach(e, p, ecsContainer); err != nil {
				log.Printf("error picking up what's on %v for entity %d: %v\n", p.Tile, e, err)
			}
		}
		if r := input.Roof; r.Work {
			ecsContainer.WorkOnRoof(e, r.Tile)
		}
		if m := input.MoveOnGround; m.Move && inItemReach(e, m.From, ecsContainer) && inItemReach(e, m.To, ecsContainer) {
			ecsContainer.MoveOnGround(m.From, m.To)
		}
		// After picking up, so that what's picked up can be put down elsewhere in the same tick
		if d := input.PutDown; d.PutDown && inItemReach(e, d.Tile, ecsContainer) {
			if _, err := ecsContainer.PutDownOn(e, d.From, d.Amount, d.Tile); err != nil {
				log.Printf("error putting slot %d of entity %d down on %v: %v\n", d.From+1, e, d.Tile, err)
			}
		}

		if err := applyInventoryAction(e, input.Inventory, ecsContainer); err != nil {
			log.Printf("error applying inventory action %+v of entity %d: %v\n", input.Inventory, e, err)
		}

		if input.DropItem && ecsContainer.Inventories.HasComponent(e) {
			if err := dropOnAim(e, input, ecsContainer); err != nil {
				log.Printf("error dropping the selected item of entity %d: %v\n", e, err)
			}
		}

		// Only walking is slowed by the ground, not what pushes or flings the entity
		if vel, ok := ecsContainer.Velocities.Get(e); ok {
			dir := utils.Vec2f{X: input.Analog1X, Y: input.Analog1Y}.Normalized()
			moving := float64(ecsContainer.Capacity(e, data.Capacity_Moving)) / data.FullEfficiency
			ground := float64(ecsContainer.GroundSpeedPercent(e)) / 100
			vel.AddForce(dir.Multiply(vel.Acceleration() * moving * ground))
		}
	}

	facingsystem.Tick(ecsContainer)

	for _, e := range entities {
		useAbilities(e, allInputs[e], ecsContainer)
	}

	return nil
}

func useAbilities(e common.EntityId, input ecs.InputState, ecsContainer *ecs.ECSContainer) {
	target := ecs.AbilityTarget{}
	target.Pos, target.HasPos = ecsContainer.AimOf(e)

	equipTriggers := []equipTrigger{
		{input.MainHandEqAbility1, ecs.Equip_MainHand, 0, "main hand ability 1"},
		{input.MainHandEqAbility2, ecs.Equip_MainHand, 1, "main hand ability 2"},
		{input.OffHandEqAbility1, ecs.Equip_OffHand, 0, "off hand ability 1"},
		{input.OffHandEqAbility2, ecs.Equip_OffHand, 1, "off hand ability 2"},
	}

	// Before what's in hand, which isn't also used on a construction site the click delivers to or builds
	constructing := false
	if input.MainHandEqAbility1 > 0 && target.HasPos {
		var err error
		if constructing, err = ecsContainer.Construct(e, target.Pos, input.All); err != nil {
			log.Printf("error constructing for entity %d: %v\n", e, err)
		}
	}

	if ecsContainer.Equippers.HasComponent(e) {
		for i, t := range equipTriggers {
			if math.Abs(t.value) <= 0 || (constructing && i == 0) {
				continue
			}
			if _, err := ecsContainer.ActivateEquipmentAbility(e, t.slot, target, t.abiIdx); err != nil {
				log.Printf("error activating %s for entity %d: %v\n", t.label, e, err)
			}
		}
	}

	selfTriggers := []selfTrigger{
		{input.Ability1, 0, "ability 1"},
		{input.Ability2, 1, "ability 2"},
	}

	if ecsContainer.Abilities.HasComponent(e) {
		for _, t := range selfTriggers {
			if math.Abs(t.value) <= 0 {
				continue
			}
			if _, err := ecsContainer.ActivateAbility(e, target, t.abiIdx); err != nil {
				log.Printf("error activating %s for entity %d: %v\n", t.label, e, err)
			}
		}
	}
}

func applyInventoryAction(e common.EntityId, action ecs.InventoryAction, ecsContainer *ecs.ECSContainer) error {
	switch action.Kind {
	case ecs.InventoryAction_Move:
		return ecsContainer.MoveInventoryItem(e, action.From, action.To)
	case ecs.InventoryAction_Bind:
		inv, ok := ecsContainer.Inventories.Get(e)
		if !ok {
			return fmt.Errorf("entity %d has no inventory", e)
		}
		item, ok := inv.Slot(action.From)
		if !ok {
			return nil
		}
		return ecsContainer.BindHotbar(e, action.To, item)
	case ecs.InventoryAction_Unbind:
		return ecsContainer.BindHotbar(e, action.To, -1)
	case ecs.InventoryAction_Wear, ecs.InventoryAction_ToggleWorn:
		inv, ok := ecsContainer.Inventories.Get(e)
		if !ok {
			return fmt.Errorf("entity %d has no inventory", e)
		}
		item, ok := inv.Slot(action.From)
		if !ok {
			return nil
		}
		if action.Kind == ecs.InventoryAction_ToggleWorn {
			return ecsContainer.ToggleWorn(e, item)
		}
		_, err := ecsContainer.Wear(e, item, ecs.EquipSlotEnum(action.To))
		return err
	case ecs.InventoryAction_TakeOff:
		return ecsContainer.TakeOff(e, ecs.EquipSlotEnum(action.To))
	}
	return nil
}

// From within reach and in sight, as building is
func pickUpInReach(e common.EntityId, p ecs.ItemPickUp, ecsContainer *ecs.ECSContainer) error {
	if !inItemReach(e, p.Tile, ecsContainer) {
		return nil
	}
	var err error
	if p.Amount > 0 {
		_, err = ecsContainer.PickUpSomeFromGround(e, p.Tile, p.Amount)
	} else {
		_, err = ecsContainer.PickUpFromGround(e, p.Tile)
	}
	return err
}

func inItemReach(e common.EntityId, t utils.Vec2i, ecsContainer *ecs.ECSContainer) bool {
	_, inReach := ecsContainer.AimedTileInReach(e, utils.TilePosToWorldPos(t), data.ItemReachTiles)
	return inReach
}

// On the aimed tile while it's within reach and in sight, and otherwise where the entity stands
func dropOnAim(e common.EntityId, input ecs.InputState, ecsContainer *ecs.ECSContainer) error {
	pos, err := ecsContainer.WorldPos(e)
	if err != nil {
		return err
	}
	t := utils.WorldPosToTilePos(pos)
	if aimed, inReach := ecsContainer.AimedTileInReach(e, input.AimPos, data.ItemReachTiles); input.HasAim && inReach {
		t = aimed
	}
	_, err = ecsContainer.DropSelected(e, t)
	return err
}
