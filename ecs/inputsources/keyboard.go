package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func KeyboardInputSource(
	e common.EntityId,
	tick uint64,
	world *ecs.World,
) ecs.InputState {
	im := ecs.InputManager{}
	is := ecs.InputState{}

	config, err := im.GetInputConfig(e, world)
	if err != nil {
		log.Printf("error getting input config for entity %d: %v\n", e, err)
		return is
	}

	if ebiten.IsKeyPressed(config.Left) {
		is.Analog1X = -1.0
	}
	if ebiten.IsKeyPressed(config.Right) {
		is.Analog1X = 1.0
	}
	if ebiten.IsKeyPressed(config.Up) {
		is.Analog1Y = -1.0
	}
	if ebiten.IsKeyPressed(config.Down) {
		is.Analog1Y = 1.0
	}

	return is
}
