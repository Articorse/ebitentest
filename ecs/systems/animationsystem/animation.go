package animationsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(world *ecs.World) error {
	am := ecs.AnimationManager{}
	sm := ecs.SpriteManager{}

	for e, _ := range world.Animations {
		err := am.Tick(e, world.Animations)
		if err != nil {
			log.Printf("Error ticking animation for entity %d: %v\n", e, err)
			continue
		}

		_, ok := world.Sprites[e]
		if !ok {
			return &common.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Animation",
				MissingComponent: "Sprite",
			}
		}

		currentFrame, err := am.GetCurrentFrame(e, world.Animations)
		if err != nil {
			log.Printf("Error getting current frame for entity %d: %v\n", e, err)
			continue
		}

		err = sm.SetImage(e, currentFrame, world.Sprites)
		if err != nil {
			log.Printf("Error setting sprite image for entity %d: %v\n", e, err)
			continue
		}

		layerYOffset := utils.GetFirstOpaquePixelY(currentFrame)

		err = sm.SetLocalLayerYOffset(e, layerYOffset, world.Sprites)
		if err != nil {
			log.Printf("Error setting local layer Y offset for entity %d: %v\n", e, err)
			continue
		}
	}

	for e, _ := range world.Sprites {
		err := sm.TickFlash(e, world.Sprites)
		if err != nil {
			log.Printf("Error ticking flash for entity %d: %v\n", e, err)
			continue
		}
	}

	return nil
}
