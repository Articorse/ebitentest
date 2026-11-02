package effects

import (
	"ebittest/content/timerfuncs"
	"ebittest/ecs"
	"ebittest/utils"
	"fmt"
	"image/color"
)

func SpawnDamageNumber(pos utils.Vec2f, damage int, ecsContainer *ecs.ECSContainer) error {
	timerComp, err := ecs.NewTimerComponent(1000, 1, timerfuncs.TimerFunc_Selfdestruct)
	if err != nil {
		return fmt.Errorf("error creating damage number timer component: %w", err)
	}

	_, err = ecsContainer.AddNewEntity(
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewFloatingTextComponent(fmt.Sprintf("%d", damage), utils.Vec2f{}, 12, color.RGBA{R: 255, G: 0, B: 255, A: 255}),
		ecs.NewVelocityComponentWithParams(utils.Vec2f{X: 0, Y: -1}, 1, 1),
		timerComp,
	)
	if err != nil {
		return fmt.Errorf("error spawning damage number at %v: %w", pos, err)
	}

	return nil
}
