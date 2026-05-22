package components

import (
	"ebittest/ecs/ecscommon"

	"github.com/hajimehoshi/ebiten/v2"
)

type Input struct {
	Up, Down, Left, Right ebiten.Key
	Use                   ebiten.MouseButton
	InputSourceFunc       ecscommon.InputSourceFunc
}
