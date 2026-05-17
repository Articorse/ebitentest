package movementsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"math"
)

// Should be called before other systems modify Transforms or Velocities
func TickEarly(
	velocities map[ecscommon.EntityId]*components.Velocity,
	transforms map[ecscommon.EntityId]*components.Transform,
) error {
	for e, velComp := range velocities {
		traComp, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Velocity",
				MissingComponent: "Transform",
			}
		}

		traComp.SetPos(traComp.GetPos().Add(velComp.Vector))
		velComp.Vector = velComp.Vector.Multiply(velComp.Drag)

		if math.Abs(velComp.Vector.X) < data.VelocityThreshold {
			velComp.Vector.X = 0
		}

		if math.Abs(velComp.Vector.Y) < data.VelocityThreshold {
			velComp.Vector.Y = 0
		}
	}

	return nil
}

// Should be called after other systems modify Transforms or Velocities
func TickLate(
	transforms map[ecscommon.EntityId]*components.Transform,
) error {
	for _, traComp := range transforms {
		traComp.SetPrevPos(traComp.GetPos())
	}

	return nil
}
