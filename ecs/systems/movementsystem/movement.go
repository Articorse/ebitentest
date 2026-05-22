package movementsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"log"
)

// Should be called before other systems modify Transforms or Velocities
func TickEarly(
	velocities map[ecscommon.EntityId]*components.Velocity,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	tm := components.TransformManager{}
	vm := components.VelocityManager{}

	for e, _ := range velocities {
		localPos, err := tm.GetLocalPos(e, transforms)
		if err != nil {
			return fmt.Errorf("error getting local position of entity %d: %v", e, err)
		}

		drag, err := vm.GetDrag(e, velocities)
		if err != nil {
			return fmt.Errorf("error getting drag of entity %d: %v", e, err)
		}

		localVelVec, err := vm.GetLocalVector(e, velocities)
		if err != nil {
			return fmt.Errorf("error getting local velocity vector of entity %d: %v", e, err)
		}

		tm.SetLocalPos(e, localPos.Add(localVelVec), transforms)
		vm.SetLocalVector(e, localVelVec.Multiply(drag), velocities)

		if localVelVec.Length() < data.VelocityThreshold {
			vm.SetLocalVector(e, utils.Vec2{X: 0, Y: 0}, velocities)
		}
	}

	return nil
}

// Should be called after other systems modify Transforms or Velocities
func TickLate(
	transforms map[ecscommon.EntityId]*components.Transform,
) error {
	tm := components.TransformManager{}

	for e, _ := range transforms {
		localPrevPos, err := tm.GetLocalPos(e, transforms)
		if err != nil {
			log.Printf("error getting local previous position of root entity: %v\n", err)
			continue
		}

		err = tm.SetLocalPrevPos(e, localPrevPos, transforms)
		if err != nil {
			log.Printf("error setting local position of root entity: %v\n", err)
			continue
		}
	}

	return nil
}
