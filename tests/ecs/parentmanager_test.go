package ecs_test

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"testing"
)

func TestGetOrderedHierarchies(t *testing.T) {
	entities := []common.EntityId{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}
	world := &ecs.World{}

	world.AddComponent(0, ecs.NewParentComponent())
	world.AddComponent(1, ecs.NewParentComponent())
	world.AddComponent(2, ecs.NewParentComponent())
	world.AddComponent(3, ecs.NewParentComponent())
	world.AddComponent(4, ecs.NewParentComponent())
	world.AddComponent(5, ecs.NewParentComponent())
	world.AddComponent(6, ecs.NewParentComponent())
	world.AddComponent(7, ecs.NewParentComponent())
	world.AddComponent(8, ecs.NewParentComponent())
	world.AddComponent(9, ecs.NewParentComponent())
	world.AddComponent(10, ecs.NewParentComponent())
	world.AddComponent(11, ecs.NewParentComponent())
	world.AddComponent(12, ecs.NewParentComponent())
	world.AddComponent(13, ecs.NewParentComponent())
	world.AddComponent(14, ecs.NewParentComponent())
	world.AddComponent(15, ecs.NewParentComponent())
	world.AddComponent(16, ecs.NewParentComponent())
	world.AddComponent(17, ecs.NewParentComponent())
	world.AddComponent(18, ecs.NewParentComponent())
	world.AddComponent(19, ecs.NewParentComponent())
	world.AddComponent(20, ecs.NewParentComponent())
	world.AddComponent(21, ecs.NewParentComponent())
	world.AddComponent(22, ecs.NewParentComponent())
	world.AddComponent(23, ecs.NewParentComponent())
	world.AddComponent(24, ecs.NewParentComponent())
	world.AddComponent(25, ecs.NewParentComponent())

	world.AddComponent(0, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(1, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(2, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(3, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(4, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(5, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(6, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(7, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(8, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(9, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(10, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(11, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(12, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(13, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(14, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(15, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(16, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(17, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(18, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(19, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(20, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(21, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(22, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(23, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(24, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))
	world.AddComponent(25, ecs.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0))

	pm := ecs.ParentManager{}

	pm.Attach(14, 12, world)
	pm.Attach(13, 9, world)
	pm.Attach(12, 8, world)
	pm.Attach(11, 8, world)
	pm.Attach(10, 7, world)
	pm.Attach(9, 6, world)
	pm.Attach(8, 5, world)
	pm.Attach(7, 5, world)
	pm.Attach(6, 3, world)
	pm.Attach(5, 2, world)
	pm.Attach(4, 1, world)
	pm.Attach(3, 1, world)
	pm.Attach(2, 0, world)
	pm.Attach(1, 0, world)

	pm.Attach(21, 19, world)
	pm.Attach(20, 17, world)
	pm.Attach(19, 16, world)
	pm.Attach(18, 16, world)
	pm.Attach(17, 16, world)
	pm.Attach(16, 15, world)

	pm.Attach(22, 21, world)
	pm.Attach(23, 21, world)
	pm.Attach(24, 22, world)
	pm.Attach(15, 25, world)

	hierarchies, err := pm.GetOrderedHierarchies(entities, world)
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
