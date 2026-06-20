package animationsystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

func Tick(ecs *ecs.ECS) error {
	am := ecs.AnimationManager
	sm := ecs.SpriteManager

	for _, e := range ecs.Animations.GetEntities() {
		err := am.Tick(e, ecs)
		if err != nil {
			log.Printf("Error ticking animation for entity %d: %v\n", e, err)
			continue
		}

		// TODO: Don't halt all animation processing if one fails
		if !ecs.Sprites.HasComponent(e) {
			return &common.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Animation",
				MissingComponent: "Sprite",
			}
		}

		currentFrame, err := am.GetCurrentFrame(e, ecs)
		if err != nil {
			log.Printf("Error getting current frame for entity %d: %v\n", e, err)
			continue
		}

		err = sm.SetImage(e, currentFrame, ecs)
		if err != nil {
			log.Printf("Error setting sprite image for entity %d: %v\n", e, err)
			continue
		}

		// TODO: Only do this if the image has changed.
		// Maybe calculate these once for each frame and store them in the animation component.
		layerYOffset := utils.GetFirstOpaquePixelY(currentFrame)

		err = sm.SetLocalLayerYOffset(e, layerYOffset, ecs)
		if err != nil {
			log.Printf("Error setting local layer Y offset for entity %d: %v\n", e, err)
			continue
		}
	}

	for _, e := range ecs.Sprites.GetEntities() {
		err := sm.TickFlash(e, ecs)
		if err != nil {
			log.Printf("Error ticking flash for entity %d: %v\n", e, err)
			continue
		}
	}

	return nil
}
