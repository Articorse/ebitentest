package inputsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"
)

func GetTickInputs(
	inputs map[ecscommon.EntityId]*components.Input,
	tick uint64,
	inputSource ecscommon.InputSourceFunc,
) map[ecscommon.EntityId]ecscommon.InputState {
	tickInputs := make(map[ecscommon.EntityId]ecscommon.InputState)
	for e := range inputs {
		input := inputSource(e, tick)
		tickInputs[e] = input
	}
	return tickInputs
}

func HandleInputs(
	velocities map[ecscommon.EntityId]*components.Velocity,
	allInputs map[ecscommon.EntityId]ecscommon.InputState,
) error {
	vm := components.VelocityManager{}

	for e, input := range allInputs {
		_, hasVel := velocities[e]
		if !hasVel {
			continue
		}

		v := utils.Vec2{X: 0, Y: 0}

		if input.Left {
			v.X -= 1
		}
		if input.Right {
			v.X += 1
		}
		if input.Up {
			v.Y -= 1
		}
		if input.Down {
			v.Y += 1
		}

		v = v.Normalized()

		pLocalVelVec, err := vm.GetLocalVector(e, velocities)
		if err != nil {
			log.Printf("Error getting local velocity vector for entity %d: %v\n", e, err)
			continue
		}

		pAccel, err := vm.GetAcceleration(e, velocities)
		if err != nil {
			log.Printf("Error getting acceleration for entity %d: %v\n", e, err)
			continue
		}

		vm.SetLocalVector(e, pLocalVelVec.Add(v.Multiply(pAccel)), velocities)
	}

	return nil
}
