package drawsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

// TODO: Iterate over sprites based on Layer,
// then use SHG to make slices of entities based on cell position,
// then sort each slice on their Y position + LayerYOffset,
// then draw
func DrawFrame(
	screen *ebiten.Image,
	camera utils.Vec2,
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*components.Sprite,
	transforms map[ecscommon.EntityId]*components.Transform,
) error {
	for e, sprComp := range sprites {
		traComp, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponent{
				Entity:           e,
				PresentComponent: "Sprite",
				MissingComponent: "Transform",
			}
		}

		if sprComp.Image == nil {
			continue
		}

		v := traComp.Pos.Add(sprComp.OffsetPos)
		r := traComp.Rotation + sprComp.OffsetRotation
		s := traComp.Scale * sprComp.OffsetScale

		opts := ebiten.DrawImageOptions{}
		w, h := sprComp.Image.Bounds().Dx(), sprComp.Image.Bounds().Dy()
		opts.GeoM.Scale(s, s)
		opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
		opts.GeoM.Rotate(r)
		opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

		screen.DrawImage(sprComp.Image, &opts)
	}

	return nil
}
