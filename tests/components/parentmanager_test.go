package components_test

import (
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"testing"
)

func TestGetOrderedHierarchies(t *testing.T) {
	entities := []ecscommon.EntityId{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}

	parents := map[ecscommon.EntityId]*components.Parent{
		0:  components.NewParentComponent(),
		1:  components.NewParentComponent(),
		2:  components.NewParentComponent(),
		3:  components.NewParentComponent(),
		4:  components.NewParentComponent(),
		5:  components.NewParentComponent(),
		6:  components.NewParentComponent(),
		7:  components.NewParentComponent(),
		8:  components.NewParentComponent(),
		9:  components.NewParentComponent(),
		10: components.NewParentComponent(),
		11: components.NewParentComponent(),
		12: components.NewParentComponent(),
		13: components.NewParentComponent(),
		14: components.NewParentComponent(),
		15: components.NewParentComponent(),
		16: components.NewParentComponent(),
		17: components.NewParentComponent(),
		18: components.NewParentComponent(),
		19: components.NewParentComponent(),
		20: components.NewParentComponent(),
		21: components.NewParentComponent(),
		22: components.NewParentComponent(),
		23: components.NewParentComponent(),
		24: components.NewParentComponent(),
		25: components.NewParentComponent(),
	}

	transforms := map[ecscommon.EntityId]*components.Transform{
		0:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		1:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		2:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		3:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		4:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		5:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		6:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		7:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		8:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		9:  components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		10: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		11: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		12: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		13: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		14: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		15: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		16: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		17: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		18: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		19: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		20: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		21: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		22: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		23: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		24: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
		25: components.NewTransformComponent(utils.Vec2{X: 0, Y: 0}, 1, 0),
	}

	pm := components.ParentManager{}

	pm.Attach(14, 12, transforms, parents)
	pm.Attach(13, 9, transforms, parents)
	pm.Attach(12, 8, transforms, parents)
	pm.Attach(11, 8, transforms, parents)
	pm.Attach(10, 7, transforms, parents)
	pm.Attach(9, 6, transforms, parents)
	pm.Attach(8, 5, transforms, parents)
	pm.Attach(7, 5, transforms, parents)
	pm.Attach(6, 3, transforms, parents)
	pm.Attach(5, 2, transforms, parents)
	pm.Attach(4, 1, transforms, parents)
	pm.Attach(3, 1, transforms, parents)
	pm.Attach(2, 0, transforms, parents)
	pm.Attach(1, 0, transforms, parents)

	pm.Attach(21, 19, transforms, parents)
	pm.Attach(20, 17, transforms, parents)
	pm.Attach(19, 16, transforms, parents)
	pm.Attach(18, 16, transforms, parents)
	pm.Attach(17, 16, transforms, parents)
	pm.Attach(16, 15, transforms, parents)

	pm.Attach(22, 21, transforms, parents)
	pm.Attach(23, 21, transforms, parents)
	pm.Attach(24, 22, transforms, parents)
	pm.Attach(15, 25, transforms, parents)

	hierarchies, err := pm.GetOrderedHierarchies(entities, parents)
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
