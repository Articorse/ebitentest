package inputsources

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func KeyboardMouseInputSource(
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

	mX, mY := ebiten.CursorPosition()
	is.MousePos = utils.Vec2{X: float64(mX), Y: float64(mY)}
	if inpututil.IsMouseButtonJustPressed(config.Use) {
		is.Use = true
		log.Println("use key just pressed")
	}

	return is
}
