package inputsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

func GetTickInputs(
	playerInputs map[ecscommon.PlayerId]*ecscommon.InputConfig,
	tick uint64,
	inputSource ecscommon.InputSourceFunc,
) map[ecscommon.PlayerId]ecscommon.InputState {
	tickInputs := make(map[ecscommon.PlayerId]ecscommon.InputState)
	for playerId := range playerInputs {
		input := inputSource(playerId, tick)
		tickInputs[playerId] = input
	}
	return tickInputs
}

func HandleInputs(w *ecs.World, allInputs map[ecscommon.PlayerId]ecscommon.InputState) error {
	for playerId, input := range allInputs {
		pE, ok := w.PlayerEntities[playerId]
		if !ok {
			return fmt.Errorf("player %s does not have an associated entity: %v", playerId)
		}

		pVelComp, hasVel := w.Velocities[pE]

		if hasVel {
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
			pVelComp.Vector = pVelComp.Vector.Add(v.Multiply(pVelComp.Acceleration))
		}
	}

	return nil
}
