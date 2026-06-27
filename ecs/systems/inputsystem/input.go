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
	camera utils.Vec2,
	ecsContainer *ecs.ECSContainer,
	allInputs map[common.EntityId]ecs.InputState,
) error {
	for e, input := range allInputs {
		if ecsContainer.Transforms.HasComponent(e) {
			tm := ecsContainer.TransformManager
			fpm := ecsContainer.FacePositionManager

			eWorldPos, err := tm.GetWorldPos(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", e, err)
				continue
			}

			hasFPComp := ecsContainer.FacePositions.HasComponent(e)

			if hasFPComp {
				hpEnabled, err := fpm.GetEnabled(e, ecsContainer)
				if err != nil {
					log.Printf("Error getting face position enabled for entity %d: %v\n", e, err)
					continue
				}

				if hpEnabled {
					mX := input.FacingDir.X
					mY := input.FacingDir.Y

					if mX != 0 || mY != 0 {
						dX := mX - eWorldPos.X
						dY := mY - eWorldPos.Y
						r := math.Atan2(dY, dX)

						err = tm.SetLocalRotation(e, r, ecsContainer)
						if err != nil {
							log.Printf("Error setting world rotation for entity %d: %v\n", e, err)
						}
					}
				}
			}

			if ecsContainer.Equippers.HasComponent(e) {
				eqm := ecsContainer.EquipManager

				eqEntities, err := eqm.GetEquipmentEntities(e, ecsContainer)
				if err != nil {
					log.Printf("Error getting equipment entities for entity %d: %v\n", e, err)
					continue
				}

				for _, eqE := range eqEntities {
					hasFPComp := ecsContainer.FacePositions.HasComponent(eqE)

					if hasFPComp {
						hpEnabled, err := fpm.GetEnabled(eqE, ecsContainer)
						if err != nil {
							log.Printf("Error getting face position enabled for entity %d: %v\n", eqE, err)
							continue
						}

						if hpEnabled {
							mX := input.FacingDir.X
							mY := input.FacingDir.Y

							if mX != 0 || mY != 0 {
								dX := mX - eWorldPos.X
								dY := mY - eWorldPos.Y
								r := math.Atan2(dY, dX)

								err = tm.SetLocalRotation(eqE, r, ecsContainer)
								if err != nil {
									log.Printf("Error setting world rotation for entity %d: %v\n", eqE, err)
								}
							}
						}
					}
				}
			}
		}

		if ecsContainer.Velocities.HasComponent(e) {
			vm := ecsContainer.VelocityManager

			v := utils.Vec2{X: 0, Y: 0}

			v.X = input.Analog1X
			v.Y = input.Analog1Y

			v = v.Normalized()

			pLocalVelVec, err := vm.GetLocalVector(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting local velocity vector for entity %d: %v\n", e, err)
				continue
			}

			pAccel, err := vm.GetAcceleration(e, ecsContainer)
			if err != nil {
				log.Printf("Error getting acceleration for entity %d: %v\n", e, err)
				continue
			}

			err = vm.SetLocalVector(e, pLocalVelVec.Add(v.Multiply(pAccel)), ecsContainer)
			if err != nil {
				log.Printf("Error setting local velocity vector for entity %d: %v\n", e, err)
				continue
			}
		}

		equipTriggers := []equipTrigger{
			{input.MainHandEqAbility1, ecs.Equip_MainHand, 0, "main hand ability 1"},
			{input.MainHandEqAbility2, ecs.Equip_MainHand, 1, "main hand ability 2"},
			{input.OffHandEqAbility1, ecs.Equip_OffHand, 0, "off hand ability 1"},
			{input.OffHandEqAbility2, ecs.Equip_OffHand, 1, "off hand ability 2"},
		}

		if ecsContainer.Equippers.HasComponent(e) {
			em := ecsContainer.EquipManager
			for _, t := range equipTriggers {
				if math.Abs(t.value) <= 0 {
					continue
				}
				if _, err := em.ActivateAbility(e, t.slot, nil, utils.Vec2{}, t.abiIdx, ecsContainer); err != nil {
					log.Printf("error activating %s for entity %d: %v\n", t.label, e, err)
				}
			}
		}

		selfTriggers := []selfTrigger{
			{input.Ability1, 0, "ability 1"},
			{input.Ability2, 1, "ability 2"},
		}

		if ecsContainer.Abilities.HasComponent(e) {
			am := ecsContainer.AbilitiesManager
			for _, t := range selfTriggers {
				if math.Abs(t.value) <= 0 {
					continue
				}
				if _, err := am.ActivateAbility(e, nil, utils.Vec2{}, t.abiIdx, ecsContainer); err != nil {
					log.Printf("error activating %s for entity %d: %v\n", t.label, e, err)
				}
			}
		}
	}

	return nil
}
