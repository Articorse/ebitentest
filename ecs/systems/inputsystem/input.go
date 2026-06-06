package inputsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/spawnersystem"
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
	for e := range world.Inputs {
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
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	for e, input := range allInputs {
		_, hasTra := world.Transforms[e]
		if hasTra {
			eWorldPos, err := tm.GetWorldPos(e, world.Transforms, world.Parents)
			if err != nil {
				log.Printf("Error getting world position for entity %d: %v\n", e, err)
				continue
			}

			mX := input.MouseScreenPos.X
			mY := input.MouseScreenPos.Y

			if mX != 0 || mY != 0 {
				mWorldX := mX + camera.X
				mWorldY := mY + camera.Y
				dX := mWorldX - eWorldPos.X
				dY := mWorldY - eWorldPos.Y
				r := math.Atan2(dY, dX)

				err = tm.SetLocalRotation(e, r, world.Transforms)
				if err != nil {
					log.Printf("Error setting world rotation for entity %d: %v\n", e, err)
				}
			}
		}

		_, hasVel := world.Velocities[e]
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

			pLocalVelVec, err := vm.GetLocalVector(e, world.Velocities)
			if err != nil {
				log.Printf("Error getting local velocity vector for entity %d: %v\n", e, err)
				continue
			}

			pAccel, err := vm.GetAcceleration(e, world.Velocities)
			if err != nil {
				log.Printf("Error getting acceleration for entity %d: %v\n", e, err)
				continue
			}

			vm.SetLocalVector(e, pLocalVelVec.Add(v.Multiply(pAccel)), world.Velocities)
		}

		_, hasSpawner := world.Spawners[e]
		if hasSpawner {
			if input.Use {
				spawnersystem.Spawn(e, world)
			}
		}
	}

	return nil
}
