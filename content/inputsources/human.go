package inputsources

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func HumanInputSource(
	e common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	in, ok := ecsContainer.Inputs.Get(e)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("entity %d has no input", e)
	}

	is := ecs.InputState{
		Analog1Y:           in.Value(ecs.Input_Analog1Y),
		Analog1X:           in.Value(ecs.Input_Analog1X),
		Analog2Y:           in.Value(ecs.Input_Analog2Y),
		Analog2X:           in.Value(ecs.Input_Analog2X),
		MainHandEqAbility1: in.Value(ecs.Input_MainHandAbility1),
		MainHandEqAbility2: in.Value(ecs.Input_MainHandAbility2),
		OffHandEqAbility1:  in.Value(ecs.Input_OffHandAbility1),
		OffHandEqAbility2:  in.Value(ecs.Input_OffHandAbility2),
		Ability1:           in.Value(ecs.Input_Ability1),
		Ability2:           in.Value(ecs.Input_Ability2),
	}

	lastFacingDir := in.LastFacingDir()
	worldPos, err := ecsContainer.WorldPos(e)

	var mX, mY float64
	switch in.FacingInput() {
	case ecs.Facing_None:

	case ecs.Facing_Mouse:
		mXint, mYint := ebiten.CursorPosition()
		mX = float64(mXint) + ecsContainer.Camera.X
		mY = float64(mYint) + ecsContainer.Camera.Y
		is.FacingDir = utils.Vec2f{X: float64(mX), Y: float64(mY)}

	case ecs.Facing_Analog2:
		if err != nil {
			log.Printf("Error getting world position for entity %d: %v\n", e, err)
			break
		}
		mVec := utils.Vec2f{X: is.Analog2X, Y: is.Analog2Y}
		if mVec.Length() > data.GamepadAimDeadzone {
			in.SetLastFacingDir(utils.Vec2f{X: is.Analog2X, Y: is.Analog2Y})
		}
		mX = lastFacingDir.X + worldPos.X
		mY = lastFacingDir.Y + worldPos.Y
		is.FacingDir = utils.Vec2f{X: mX, Y: mY}

	default:
		log.Printf("Unknown facing input %d for entity %d\n", in.FacingInput(), e)
	}

	return is, err
}
