package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"testing"
)

func TestGetOrderedHierarchies(t *testing.T) {
	entities := []common.EntityId{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}
	ecsContainer := ecs.NewECSContainer()

	ecsContainer.AddComponent(0, ecs.NewParentComponent())
	ecsContainer.AddComponent(1, ecs.NewParentComponent())
	ecsContainer.AddComponent(2, ecs.NewParentComponent())
	ecsContainer.AddComponent(3, ecs.NewParentComponent())
	ecsContainer.AddComponent(4, ecs.NewParentComponent())
	ecsContainer.AddComponent(5, ecs.NewParentComponent())
	ecsContainer.AddComponent(6, ecs.NewParentComponent())
	ecsContainer.AddComponent(7, ecs.NewParentComponent())
	ecsContainer.AddComponent(8, ecs.NewParentComponent())
	ecsContainer.AddComponent(9, ecs.NewParentComponent())
	ecsContainer.AddComponent(10, ecs.NewParentComponent())
	ecsContainer.AddComponent(11, ecs.NewParentComponent())
	ecsContainer.AddComponent(12, ecs.NewParentComponent())
	ecsContainer.AddComponent(13, ecs.NewParentComponent())
	ecsContainer.AddComponent(14, ecs.NewParentComponent())
	ecsContainer.AddComponent(15, ecs.NewParentComponent())
	ecsContainer.AddComponent(16, ecs.NewParentComponent())
	ecsContainer.AddComponent(17, ecs.NewParentComponent())
	ecsContainer.AddComponent(18, ecs.NewParentComponent())
	ecsContainer.AddComponent(19, ecs.NewParentComponent())
	ecsContainer.AddComponent(20, ecs.NewParentComponent())
	ecsContainer.AddComponent(21, ecs.NewParentComponent())
	ecsContainer.AddComponent(22, ecs.NewParentComponent())
	ecsContainer.AddComponent(23, ecs.NewParentComponent())
	ecsContainer.AddComponent(24, ecs.NewParentComponent())
	ecsContainer.AddComponent(25, ecs.NewParentComponent())

	ecsContainer.AddComponent(0, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(1, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(2, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(3, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(4, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(5, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(6, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(7, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(8, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(9, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(10, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(11, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(12, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(13, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(14, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(15, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(16, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(17, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(18, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(19, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(20, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(21, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(22, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(23, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(24, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))
	ecsContainer.AddComponent(25, ecs.NewTransformComponent(utils.Vec2f{X: 0, Y: 0}, 1, 0))

	pm := ecsContainer.ParentManager

	pm.Attach(14, 12, ecsContainer)
	pm.Attach(13, 9, ecsContainer)
	pm.Attach(12, 8, ecsContainer)
	pm.Attach(11, 8, ecsContainer)
	pm.Attach(10, 7, ecsContainer)
	pm.Attach(9, 6, ecsContainer)
	pm.Attach(8, 5, ecsContainer)
	pm.Attach(7, 5, ecsContainer)
	pm.Attach(6, 3, ecsContainer)
	pm.Attach(5, 2, ecsContainer)
	pm.Attach(4, 1, ecsContainer)
	pm.Attach(3, 1, ecsContainer)
	pm.Attach(2, 0, ecsContainer)
	pm.Attach(1, 0, ecsContainer)

	pm.Attach(21, 19, ecsContainer)
	pm.Attach(20, 17, ecsContainer)
	pm.Attach(19, 16, ecsContainer)
	pm.Attach(18, 16, ecsContainer)
	pm.Attach(17, 16, ecsContainer)
	pm.Attach(16, 15, ecsContainer)

	pm.Attach(22, 21, ecsContainer)
	pm.Attach(23, 21, ecsContainer)
	pm.Attach(24, 22, ecsContainer)
	pm.Attach(15, 25, ecsContainer)

	hierarchies, err := pm.GetOrderedHierarchies(entities, ecsContainer)
	if err != nil {
		t.Errorf("error getting ordered hierarchies: %v", err)
	}

	if len(hierarchies) != 2 {
		t.Errorf("expected at least 2 hierarchy, got %d", len(hierarchies))
	}

	if len(hierarchies[0]) != 6 {
		t.Errorf("expected hierarchy 0 of length 6, got %d", len(hierarchies[0]))
	}

	if len(hierarchies[0][0]) != 1 {
		t.Errorf("expected first level of hierarchy 0 to have 1 entity, got %d", len(hierarchies[0][0]))
	}

	if len(hierarchies[0][1]) != 2 {
		t.Errorf("expected second level of hierarchy 0 to have 2 entity, got %d", len(hierarchies[0][1]))
	}

	if len(hierarchies[0][2]) != 3 {
		t.Errorf("expected third level of hierarchy 0 to have 3 entity, got %d", len(hierarchies[0][2]))
	}

	if len(hierarchies[0][3]) != 3 {
		t.Errorf("expected fourth level of hierarchy 0 to have 3 entity, got %d", len(hierarchies[0][3]))
	}

	if len(hierarchies[0][4]) != 4 {
		t.Errorf("expected fifth level of hierarchy 0 to have 4 entity, got %d", len(hierarchies[0][4]))
	}

	if len(hierarchies[0][5]) != 2 {
		t.Errorf("expected sixth level of hierarchy 0 to have 2 entity, got %d", len(hierarchies[0][5]))
	}

	if len(hierarchies[1]) != 4 {
		t.Errorf("expected hierarchy 1 of length 4, got %d", len(hierarchies[1]))
	}

	if len(hierarchies[1][0]) != 1 {
		t.Errorf("expected first level of hierarchy 1 to have 1 entity, got %d", len(hierarchies[1][0]))
	}

	if len(hierarchies[1][1]) != 1 {
		t.Errorf("expected second level of hierarchy 1 to have 1 entity, got %d", len(hierarchies[1][1]))
	}

	if len(hierarchies[1][2]) != 3 {
		t.Errorf("expected third level of hierarchy 1 to have 3 entity, got %d", len(hierarchies[1][2]))
	}

	if len(hierarchies[1][3]) != 2 {
		t.Errorf("expected fourth level of hierarchy 1 to have 2 entity, got %d", len(hierarchies[1][3]))
	}
}
