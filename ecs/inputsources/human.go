package inputsources

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func HumanInputSource(
	e common.EntityId,
	tick uint64,
	ecsContainer *ecs.ECSContainer,
) ecs.InputState {
	var err error

	im := ecsContainer.InputManager
	is := ecs.InputState{}

	is.Analog1Y, err = im.GetInput(e, ecs.Input_Analog1Y, ecsContainer)
	if err != nil {
		log.Printf("Error getting vertical axis input for entity %d: %v\n", e, err)
	}

	is.Analog1X, err = im.GetInput(e, ecs.Input_Analog1X, ecsContainer)
	if err != nil {
		log.Printf("Error getting horizontal axis input for entity %d: %v\n", e, err)
	}

	is.Analog2Y, err = im.GetInput(e, ecs.Input_Analog2Y, ecsContainer)
	if err != nil {
		log.Printf("Error getting vertical axis 2 input for entity %d: %v\n", e, err)
	}

	is.Analog2X, err = im.GetInput(e, ecs.Input_Analog2X, ecsContainer)
	if err != nil {
		log.Printf("Error getting horizontal axis 2 input for entity %d: %v\n", e, err)
	}

	is.MainHandEqAbility1, err = im.GetInput(e, ecs.Input_MainHandAbility1, ecsContainer)
	if err != nil {
		log.Printf("Error getting main hand equipment ability input for entity %d: %v\n", e, err)
	}

	is.MainHandEqAbility2, err = im.GetInput(e, ecs.Input_MainHandAbility2, ecsContainer)
	if err != nil {
		log.Printf("Error getting main hand equipment ability 2 input for entity %d: %v\n", e, err)
	}

	is.OffHandEqAbility1, err = im.GetInput(e, ecs.Input_OffHandAbility1, ecsContainer)
	if err != nil {
		log.Printf("Error getting off hand equipment ability input for entity %d: %v\n", e, err)
	}

	is.OffHandEqAbility2, err = im.GetInput(e, ecs.Input_OffHandAbility2, ecsContainer)
	if err != nil {
		log.Printf("Error getting off hand equipment ability 2 input for entity %d: %v\n", e, err)
	}

	is.Ability1, err = im.GetInput(e, ecs.Input_Ability1, ecsContainer)
	if err != nil {
		log.Printf("Error getting ability 1 input for entity %d: %v\n", e, err)
	}

	is.Ability2, err = im.GetInput(e, ecs.Input_Ability2, ecsContainer)
	if err != nil {
		log.Printf("Error getting ability 2 input for entity %d: %v\n", e, err)
	}

	tm := ecsContainer.TransformManager

	facingInput, err := im.GetFacingInput(e, ecsContainer)
	if err != nil {
		log.Printf("Error getting facing input for entity %d: %v\n", e, err)
	}

	lastFacingDir, err := im.GetLastFacingDir(e, ecsContainer)
	if err != nil {
		log.Printf("Error getting last facing direction for entity %d: %v\n", e, err)
		return is
	}

	worldPos, err := tm.GetWorldPos(e, ecsContainer)

	var mX, mY float64
	switch facingInput {
	case ecs.Facing_None:

	case ecs.Facing_Mouse:
		mXint, mYint := ebiten.CursorPosition()
		mX = float64(mXint) + ecsContainer.Camera.X
		mY = float64(mYint) + ecsContainer.Camera.Y
		is.FacingDir = utils.Vec2{X: float64(mX), Y: float64(mY)}

	case ecs.Facing_Analog2:
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			break
		}
		mVec := utils.Vec2{X: is.Analog2X, Y: is.Analog2Y}
		if mVec.Length() > data.GamepadAimDeadzone {
			err = im.SetLastFacingDir(e, utils.Vec2{X: is.Analog2X, Y: is.Analog2Y}, ecsContainer)
			if err != nil {
				log.Printf("Error setting last facing direction for entity %d: %v\n", e, err)
				return is
			}
		}
		mX = lastFacingDir.X + worldPos.X
		mY = lastFacingDir.Y + worldPos.Y
		is.FacingDir = utils.Vec2{X: mX, Y: mY}

	default:
		log.Printf("Unknown facing input %d for entity %d\n", facingInput, e)
	}

	return is
}
