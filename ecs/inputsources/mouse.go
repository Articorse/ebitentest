package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func MouseInputSource(
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

	mX, mY := ebiten.CursorPosition()
	is.MouseScreenPos = utils.Vec2{X: float64(mX), Y: float64(mY)}

	if inpututil.IsMouseButtonJustPressed(config.Use) {
		is.Use = true
	}

	return is
}
