package drawsystem

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"

	"github.com/hajimehoshi/ebiten/v2"
)

func DrawFrame(
	screen *ebiten.Image,
	sprites map[ecscommon.Entity]*components.Sprite,
	transforms map[ecscommon.Entity]*components.Transform,
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
		opts.GeoM.Translate(v.X, v.Y)

		screen.DrawImage(sprComp.Image, &opts)
	}

	return nil
}
