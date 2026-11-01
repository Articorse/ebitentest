package inputsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"math"
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

func HandleInputs(
	camera utils.Vec2f,
	ecsContainer *ecs.ECSContainer,
	allInputs map[common.EntityId]ecs.InputState,
) error {
	for e, input := range allInputs {
		if ecsContainer.Transforms.HasComponent(e) {
			eWorldPos, err := ecsContainer.WorldPos(e)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", e, err)
				continue
			}

			face(e, eWorldPos, input.FacingDir, ecsContainer)

			if equipper, ok := ecsContainer.Equippers.Get(e); ok {
				for _, eqE := range equipper.Equipment() {
					face(eqE, eWorldPos, input.FacingDir, ecsContainer)
				}
			}
		}

		if vel, ok := ecsContainer.Velocities.Get(e); ok {
			dir := utils.Vec2f{X: input.Analog1X, Y: input.Analog1Y}.Normalized()
			vel.AddForce(dir.Multiply(vel.Acceleration()))
		}

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
				if _, err := ecsContainer.ActivateEquipmentAbility(e, t.slot, nil, utils.Vec2f{}, t.abiIdx); err != nil {
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
				if _, err := ecsContainer.ActivateAbility(e, nil, utils.Vec2f{}, t.abiIdx); err != nil {
					log.Printf("error activating %s for entity %d: %v\n", t.label, e, err)
				}
			}
		}
	}

	return nil
}

func face(e common.EntityId, from utils.Vec2f, target utils.Vec2f, ecsContainer *ecs.ECSContainer) {
	fp, ok := ecsContainer.FacePositions.Get(e)
	if !ok || !fp.Enabled() || (target.X == 0 && target.Y == 0) {
		return
	}

	t, ok := ecsContainer.Transforms.Get(e)
	if !ok {
		log.Printf("Error setting rotation for entity %d: it has no transform\n", e)
		return
	}

	t.SetLocalRotation(math.Atan2(target.Y-from.Y, target.X-from.X))
}
