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

	for e, input := range allInputs {
		_, hasTra := world.Transforms[e]
		if hasTra {
			tm := ecs.TransformManager{}

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
			vm := ecs.VelocityManager{}

			v := utils.Vec2{X: 0, Y: 0}

			v.X = input.Analog1X
			v.Y = input.Analog1Y

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
				sm := ecs.SpawnerManager{}
				sm.Spawn(e, world)
			}
		}

		_, hasAnim := world.Animations[e]
		if hasAnim {
			am := ecs.AnimationManager{}
			if input.Use {
				nextState, err := am.GetState(e, world.Animations)
				if err != nil {
					log.Printf("Error getting animation state for entity %d: %v\n", e, err)
					continue
				}
				err = am.SetQueuedStateIfNone(e, nextState, world.Animations)
				if err != nil {
					log.Printf("Error setting queued animation state for entity %d: %v\n", e, err)
				}
				err = am.SetState(e, ecs.Anim_Use, world.Animations)
				if err != nil {
					log.Printf("Error setting animation state for entity %d: %v\n", e, err)
				}
			}
		}
	}

	return nil
}
