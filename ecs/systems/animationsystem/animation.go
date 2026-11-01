package animationsystem

import (
	"ebittest/assetmanager"
	"ebittest/ecs"
	"log"
)

func Tick(ecsContainer *ecs.ECSContainer, assetManager *assetmanager.AssetManager) error {
	for _, e := range ecsContainer.Animations.GetEntities() {
		if err := ecsContainer.TickAnimation(e); err != nil {
			log.Printf("Error ticking animation for entity %d: %v\n", e, err)
		}
	}

	for _, s := range ecs.Query1(&ecsContainer.Sprites) {
		s.TickFlash()
	}

	return nil
}
