package animationsystem

import (
	"ebittest/assetmanager"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
)

func Tick(ecsContainer *ecs.ECSContainer, assetManager *assetmanager.AssetManager) error {
	am := ecsContainer.AnimationManager
	sm := ecsContainer.SpriteManager

	for _, e := range ecsContainer.Animations.GetEntities() {
		err := am.Tick(e, ecsContainer)
		if err != nil {
			log.Printf("Error ticking animation for entity %d: %v\n", e, err)
			continue
		}

		// TODO: Don't halt all animation processing if one fails
		if !ecsContainer.Sprites.HasComponent(e) {
			return &common.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Animation",
				MissingComponent: "Sprite",
			}
		}
	}

	for _, e := range ecsContainer.Sprites.GetEntities() {
		err := sm.TickFlash(e, ecsContainer)
		if err != nil {
			log.Printf("Error ticking flash for entity %d: %v\n", e, err)
			continue
		}
	}

	return nil
}
