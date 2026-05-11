package inputsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
)

func GetTickInputs(
	players map[ecscommon.PlayerId]*ecscommon.PlayerConfig,
	tick uint64,
	inputSource ecscommon.InputSourceFunc,
) map[ecscommon.PlayerId]ecscommon.InputState {
	tickInputs := make(map[ecscommon.PlayerId]ecscommon.InputState)
	for playerId := range players {
		input := inputSource(playerId, tick)
		tickInputs[playerId] = input
	}
	return tickInputs
}

func HandleInputs(w *ecs.World, allInputs map[ecscommon.PlayerId]ecscommon.InputState) error {
	var err error
	for playerId, input := range allInputs {
		pConf, ok := w.Players[playerId]
		if !ok {
			err = fmt.Errorf("player %s not found in world: %v", playerId, err)
		}

		pE := pConf.Entity
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

	return err
}
