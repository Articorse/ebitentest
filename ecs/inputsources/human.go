package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func HumanInputSource(
	e common.EntityId,
	tick uint64,
	world *ecs.World,
) ecs.InputState {
	var err error

	im := ecs.InputManager{}
	is := ecs.InputState{}

	is.Analog1Y, err = im.GetInput(e, ecs.Input_Analog1Y, world)
	if err != nil {
		log.Printf("Error getting vertical axis input for entity %d: %v\n", e, err)
	}

	is.Analog1X, err = im.GetInput(e, ecs.Input_Analog1X, world)
	if err != nil {
		log.Printf("Error getting horizontal axis input for entity %d: %v\n", e, err)
	}

	is.Analog2Y, err = im.GetInput(e, ecs.Input_Analog2Y, world)
	if err != nil {
		log.Printf("Error getting vertical axis 2 input for entity %d: %v\n", e, err)
	}

	is.Analog2X, err = im.GetInput(e, ecs.Input_Analog2X, world)
	if err != nil {
		log.Printf("Error getting horizontal axis 2 input for entity %d: %v\n", e, err)
	}

	is.Ability1, err = im.GetInput(e, ecs.Input_Ability1, world)
	if err != nil {
		log.Printf("Error getting ability 1 input for entity %d: %v\n", e, err)
	}

	is.Ability2, err = im.GetInput(e, ecs.Input_Ability2, world)
	if err != nil {
		log.Printf("Error getting ability 2 input for entity %d: %v\n", e, err)
	}

	facingInput, err := im.GetFacingInput(e, world)
	if err != nil {
		log.Printf("Error getting facing input for entity %d: %v\n", e, err)
	}

	tm := ecs.TransformManager{}
	worldPos, err := tm.GetWorldPos(e, world)

	var mX, mY int
	switch facingInput {
	case ecs.Facing_None:
	case ecs.Facing_Mouse:
		mX, mY = ebiten.CursorPosition()
		mX += int(world.Camera.X)
		mY += int(world.Camera.Y)
	case ecs.Facing_Analog:
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			break
		}
		mX = int(is.Analog2X + worldPos.X)
		mY = int(is.Analog2Y + worldPos.Y)
	default:
		log.Printf("Unknown facing input %d for entity %d\n", facingInput, e)
	}

	is.FacingDir = utils.Vec2{X: float64(mX), Y: float64(mY)}

	return is
}
