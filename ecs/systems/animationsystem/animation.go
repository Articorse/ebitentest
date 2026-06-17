package animationsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(world *ecs.World) error {
	am := world.AnimationManager
	sm := world.SpriteManager

	for _, e := range world.Animations.GetEntities() {
		err := am.Tick(e, world)
		if err != nil {
			log.Printf("Error ticking animation for entity %d: %v\n", e, err)
			continue
		}

		// TODO: Don't halt all animation processing if one fails
		if !world.Sprites.HasComponent(e) {
			return &common.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Animation",
				MissingComponent: "Sprite",
			}
		}

		currentFrame, err := am.GetCurrentFrame(e, world)
		if err != nil {
			log.Printf("Error getting current frame for entity %d: %v\n", e, err)
			continue
		}

		err = sm.SetImage(e, currentFrame, world)
		if err != nil {
			log.Printf("Error setting sprite image for entity %d: %v\n", e, err)
			continue
		}

		// TODO: Only do this if the image has changed.
		// Maybe calculate these once for each frame and store them in the animation component.
		layerYOffset := utils.GetFirstOpaquePixelY(currentFrame)

		err = sm.SetLocalLayerYOffset(e, layerYOffset, world)
		if err != nil {
			log.Printf("Error setting local layer Y offset for entity %d: %v\n", e, err)
			continue
		}
	}

	for _, e := range world.Sprites.GetEntities() {
		err := sm.TickFlash(e, world)
		if err != nil {
			log.Printf("Error ticking flash for entity %d: %v\n", e, err)
			continue
		}
	}

	return nil
}
