package inputsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/facingsystem"
	"ebittest/utils"
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
		if inv, ok := ecsContainer.Inventories.Get(e); ok && input.SelectSlot > 0 && input.SelectSlot-1 != inv.Selected() {
			if err := ecsContainer.SelectInventorySlot(e, input.SelectSlot-1); err != nil {
				log.Printf("error selecting slot %d of entity %d: %v\n", input.SelectSlot, e, err)
			}
		}

		if aim, ok := ecsContainer.Aims.Get(e); ok {
			if input.HasAim {
				aim.SetTarget(input.AimPos)
			} else {
				aim.ClearTarget()
			}
		}

		// After taking aim, since a dropped item is tossed towards it
		if input.DropItem && ecsContainer.Inventories.HasComponent(e) {
			if _, _, err := ecsContainer.DropSelectedItem(e); err != nil {
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

	if ecsContainer.Equippers.HasComponent(e) {
		for _, t := range equipTriggers {
			if math.Abs(t.value) <= 0 {
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
