package inputsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
	"math"
)

func GetTickInputs(
	world *ecs.World,
	tick uint64,
	inputSource ecs.InputSourceFunc,
) map[common.EntityId]ecs.InputState {
	tickInputs := make(map[common.EntityId]ecs.InputState)
	for _, e := range world.Inputs.GetEntities() {
		input := inputSource(e, tick, world)
		tickInputs[e] = input
	}
	return tickInputs
}

func HandleInputs(
	camera utils.Vec2,
	world *ecs.World,
	allInputs map[common.EntityId]ecs.InputState,
) error {
	for e, input := range allInputs {
		if world.Transforms.HasComponent(e) {
			tm := ecs.TransformManager{}
			sm := ecs.SpriteManager{} // FIXME: Replace with FacePosition component

			eWorldPos, err := tm.GetWorldPos(e, world)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", e, err)
				continue
			}

			allowRot, err := sm.GetAllowRotation(e, world)
			if err != nil {
				log.Printf("Error getting allow rotation for entity %d: %v\n", e, err)
				continue
			}

			if allowRot {

				mX := input.MouseScreenPos.X
				mY := input.MouseScreenPos.Y

				if mX != 0 || mY != 0 {
					mWorldX := mX + camera.X
					mWorldY := mY + camera.Y
					dX := mWorldX - eWorldPos.X
					dY := mWorldY - eWorldPos.Y
					r := math.Atan2(dY, dX)

					err = tm.SetLocalRotation(e, r, world)
					if err != nil {
						log.Printf("Error setting world rotation for entity %d: %v\n", e, err)
					}
				}
			}
		}

		if world.Velocities.HasComponent(e) {
			vm := ecs.VelocityManager{}

			v := utils.Vec2{X: 0, Y: 0}

			v.X = input.Analog1X
			v.Y = input.Analog1Y

			v = v.Normalized()

			pLocalVelVec, err := vm.GetLocalVector(e, world)
			if err != nil {
				log.Printf("Error getting local velocity vector for entity %d: %v\n", e, err)
				continue
			}

			pAccel, err := vm.GetAcceleration(e, world)
			if err != nil {
				log.Printf("Error getting acceleration for entity %d: %v\n", e, err)
				continue
			}

			err = vm.SetLocalVector(e, pLocalVelVec.Add(v.Multiply(pAccel)), world)
			if err != nil {
				log.Printf("Error setting local velocity vector for entity %d: %v\n", e, err)
				continue
			}
		}

		if input.Ability1 > 0 {
			if world.Abilities.HasComponent(e) {
				am := ecs.AbilitiesManager{}
				_, err := am.ActivateAbility(e, []common.EntityId{}, 0, world)
				if err != nil {
					log.Printf("Error activating dodge ability for entity %d: %v\n", e, err)
				}
			}
		}

		if input.Ability2 > 0 {
			if world.Abilities.HasComponent(e) {
				am := ecs.AbilitiesManager{}
				_, err := am.ActivateAbility(e, []common.EntityId{}, 0, world)
				if err != nil {
					log.Printf("Error activating dodge ability for entity %d: %v\n", e, err)
				}
			}
		}
	}

	return nil
}
