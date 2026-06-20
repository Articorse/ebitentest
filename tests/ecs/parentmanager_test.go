package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"testing"
)

func TestGetOrderedHierarchies(t *testing.T) {
	entities := []common.EntityId{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}
	ecs := ecs.NeNewECS

	ecs.AddComponent(0, ecs.NewParentComponent())
	ecs.AddComponent(1, ecs.NewParentComponent())
	ecs.AddComponent(2, ecs.NewParentComponent())
	ecs.AddComponent(3, ecs.NewParentComponent())
	ecs.AddComponent(4, ecs.NewParentComponent())
	ecs.AddComponent(5, ecs.NewParentComponent())
	ecs.AddComponent(6, ecs.NewParentComponent())
	ecs.AddComponent(7, ecs.NewParentComponent())
	ecs.AddComponent(8, ecs.NewParentComponent())
	ecs.AddComponent(9, ecs.NewParentComponent())
	ecs.AddComponent(10, ecs.NewParentComponent())
	ecs.AddComponent(11, ecs.NewParentComponent())
	ecs.AddComponent(12, ecs.NewParentComponent())
	ecs.AddComponent(13, ecs.NewParentComponent())
	ecs.AddComponent(14, ecs.NewParentComponent())
	ecs.AddComponent(15, ecs.NewParentComponent())
	ecs.AddComponent(16, ecs.NewParentComponent())
	ecs.AddComponent(17, ecs.NewParentComponent())
	ecs.AddComponent(18, ecs.NewParentComponent())
	ecs.AddComponent(19, ecs.NewParentComponent())
	ecs.AddComponent(20, ecs.NewParentComponent())
	ecs.AddComponent(21, ecs.NewParentComponent())
	ecs.AddComponent(22, ecs.NewParentComponent())
	ecs.AddComponent(23, ecs.NewParentComponent())
	ecs.AddComponent(24, ecs.NewParentComponent())
	ecs.AddComponent(25, ecs.NewParentComponent())

	ecs.AddComponent(0, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(1, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(2, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(3, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(4, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(5, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(6, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(7, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(8, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(9, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(10, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(11, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(12, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(13, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(14, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(15, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(16, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(17, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(18, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(19, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(20, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(21, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(22, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(23, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(24, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	ecs.AddComponent(25, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))

	pm := ecs.ParentManager

	pm.Attach(14, 12, ecs)
	pm.Attach(13, 9, ecs)
	pm.Attach(12, 8, ecs)
	pm.Attach(11, 8, ecs)
	pm.Attach(10, 7, ecs)
	pm.Attach(9, 6, ecs)
	pm.Attach(8, 5, ecs)
	pm.Attach(7, 5, ecs)
	pm.Attach(6, 3, ecs)
	pm.Attach(5, 2, ecs)
	pm.Attach(4, 1, ecs)
	pm.Attach(3, 1, ecs)
	pm.Attach(2, 0, ecs)
	pm.Attach(1, 0, ecs)

	pm.Attach(21, 19, ecs)
	pm.Attach(20, 17, ecs)
	pm.Attach(19, 16, ecs)
	pm.Attach(18, 16, ecs)
	pm.Attach(17, 16, ecs)
	pm.Attach(16, 15, ecs)

	pm.Attach(22, 21, ecs)
	pm.Attach(23, 21, ecs)
	pm.Attach(24, 22, ecs)
	pm.Attach(15, 25, ecs)

	hierarchies, err := pm.GetOrderedHierarchies(entities, ecs)
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
