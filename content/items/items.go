package items

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

// Saved items and drop tables refer to item kinds by these values, so they must never change
const (
	ItemKind_Planks ecs.ItemKindEnum = 1
)

const planksMaxStack = 100

func Register(ecsContainer *ecs.ECSContainer) error {
	items := []struct {
		kind  ecs.ItemKindEnum
		spawn ecs.ItemSpawnFunc
	}{
		{ItemKind_Planks, SpawnPlanks},
	}

	for _, i := range items {
		if err := ecsContainer.RegisterItem(i.kind, i.spawn); err != nil {
			return fmt.Errorf("error registering item kind %v: %w", i.kind, err)
		}
	}

	return nil
}

func SpawnPlanks(pos utils.Vec2f, ecsContainer *ecs.ECSContainer) (common.EntityId, error) {
	sprComp, err := ecs.NewSpriteComponent(common.AssetImagePlanks, 20, false)
	if err != nil {
		return -1, err
	}

	return ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		sprComp,
		ecs.NewStackableItemComponent(ItemKind_Planks, planksMaxStack),
		ecs.NewPersistentComponent(),
	)
}
