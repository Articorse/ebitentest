package movementsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"math"
)

// Should be called before other systems modify Transforms or Velocities
func Tick(ecsContainer *ecs.ECSContainer) error {
	for _, c := range ecs.Query2(&ecsContainer.Transforms, &ecsContainer.Velocities) {
		t, v := c.A, c.B
		localVelVec := v.LocalVector()

		cos := math.Cos(t.LocalRotation())
		sin := math.Sin(t.LocalRotation())

		movementVector := utils.Vec2f{
			X: (localVelVec.X*cos - localVelVec.Y*sin),
			Y: (localVelVec.X*sin + localVelVec.Y*cos),
		}

		t.AddLocalPos(movementVector)
		v.SetLocalVector(localVelVec.Multiply(v.Drag()))

		if localVelVec.Length() < data.VelocityThreshold {
			v.SetLocalVector(utils.Vec2f{X: 0, Y: 0})
		}
	}

	return nil
}
