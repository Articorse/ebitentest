package movementsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
)

// Should be called before other systems modify Transforms or Velocities
func TickEarly(world *ecs.World) error {
	tm := ecs.TransformManager{}
	vm := ecs.VelocityManager{}

	for e, _ := range world.Velocities {
		localPos, err := tm.GetLocalPos(e, world.Transforms)
		if err != nil {
			return fmt.Errorf("error getting local position of entity %d: %v", e, err)
		}

		drag, err := vm.GetDrag(e, world.Velocities)
		if err != nil {
			return fmt.Errorf("error getting drag of entity %d: %v", e, err)
		}

		localVelVec, err := vm.GetLocalVector(e, world.Velocities)
		if err != nil {
			return fmt.Errorf("error getting local velocity vector of entity %d: %v", e, err)
		}

		localRot, err := tm.GetLocalRotation(e, world.Transforms)
		if err != nil {
			return fmt.Errorf("error getting local rotation of entity %d: %v", e, err)
		}

		cos := math.Cos(localRot)
		sin := math.Sin(localRot)

		movementVector := utils.Vec2{
			X: (localVelVec.X*cos - localVelVec.Y*sin),
			Y: (localVelVec.X*sin + localVelVec.Y*cos),
		}

		tm.SetLocalPos(e, localPos.Add(movementVector), world.Transforms)
		vm.SetLocalVector(e, localVelVec.Multiply(drag), world.Velocities)

		if localVelVec.Length() < data.VelocityThreshold {
			vm.SetLocalVector(e, utils.Vec2{X: 0, Y: 0}, world.Velocities)
		}
	}

	return nil
}

// Should be called after other systems modify Transforms or Velocities
func TickLate(world *ecs.World) error {
	tm := ecs.TransformManager{}

	for e, _ := range world.Transforms {
		localPrevPos, err := tm.GetLocalPos(e, world.Transforms)
		if err != nil {
			log.Printf("error getting local previous position of root entity: %v\n", err)
			continue
		}

		err = tm.SetLocalPrevPos(e, localPrevPos, world.Transforms)
		if err != nil {
			log.Printf("error setting local position of root entity: %v\n", err)
			continue
		}
	}

	return nil
}
