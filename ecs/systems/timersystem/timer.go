package timersystem

import (
	"ebittest/ecs"
	"ebittest/ecs/components"
	"fmt"
)

func Tick(world *ecs.World) error {
	tlm := components.TimedLifeManager{}
	for e, _ := range world.TimedLives {
		timerOver, err := tlm.TickDown(e, world.TimedLives)
		if err != nil {
			return fmt.Errorf("error ticking down timed life component of entity %d: %v", e, err)
		}

		if timerOver {
			world.RemoveEntity(e)
		}
	}

	return nil
}
