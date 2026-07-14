package movementsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"fmt"
	"math"
)

// Should be called before other systems modify Transforms or Velocities
func Tick(ecsContainer *ecs.ECSContainer) error {
	tm := ecsContainer.TransformManager
	vm := ecsContainer.VelocityManager

	for _, e := range ecsContainer.Velocities.GetEntities() {
		localPos, err := tm.GetLocalPos(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting local position of entity %d: %v", e, err)
		}

		drag, err := vm.GetDrag(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting drag of entity %d: %v", e, err)
		}

		localVelVec, err := vm.GetLocalVector(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting local velocity vector of entity %d: %v", e, err)
		}

		localRot, err := tm.GetLocalRotation(e, ecsContainer)
		if err != nil {
			return fmt.Errorf("error getting local rotation of entity %d: %v", e, err)
		}

		cos := math.Cos(localRot)
		sin := math.Sin(localRot)

		movementVector := utils.Vec2f{
			X: (localVelVec.X*cos - localVelVec.Y*sin),
			Y: (localVelVec.X*sin + localVelVec.Y*cos),
		}

		err = tm.SetLocalPos(e, localPos.Add(movementVector), ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting local position of entity %d: %v", e, err)
		}
		err = vm.SetLocalVector(e, localVelVec.Multiply(drag), ecsContainer)
		if err != nil {
			return fmt.Errorf("error setting local velocity vector of entity %d: %v", e, err)
		}

		if localVelVec.Length() < data.VelocityThreshold {
			err = vm.SetLocalVector(e, utils.Vec2f{X: 0, Y: 0}, ecsContainer)
			if err != nil {
				return fmt.Errorf("error setting local velocity vector of entity %d: %v", e, err)
			}
		}
	}

	return nil
}
