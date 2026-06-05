package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func KeyboardInputSource(
	e ecscommon.EntityId,
	tick uint64,
	inputs map[ecscommon.EntityId]*components.Input,
) components.InputState {
	im := components.InputManager{}
	is := components.InputState{}

	config, err := im.GetInputConfig(e, inputs)
	if err != nil {
		log.Printf("error getting input config for entity %d: %v\n", e, err)
		return is
	}

	if ebiten.IsKeyPressed(config.Left) {
		is.Left = true
	}
	if ebiten.IsKeyPressed(config.Right) {
		is.Right = true
	}
	if ebiten.IsKeyPressed(config.Up) {
		is.Up = true
	}
	if ebiten.IsKeyPressed(config.Down) {
		is.Down = true
	}

	return is
}
