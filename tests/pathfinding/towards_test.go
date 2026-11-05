package pathfinding_test

import (
	"ebittest/pathfinding"
	"ebittest/utils"
	"testing"
)

func TestFindingTheWayTowardsGoalsAroundATargetTakesTheCheapest(t *testing.T) {
	// The target's tile is walled off, and of the goals around it the one on this side is the cheapest
	target := utils.Vec2i{X: 10}
	g := newGrid(target)
	near, far := utils.Vec2i{X: 8}, utils.Vec2i{X: 12}
	isGoal := func(t utils.Vec2i) bool { return t == near || t == far }

	path, cost, ok := pathfinding.FindTowards(utils.Vec2i{}, target, 2, isGoal, 10000, g.speed)
	if !ok || len(path) == 0 || path[len(path)-1] != near {
		t.Fatalf("found %v (%v), want the way to the goal on this side", path, ok)
	}
	if cost != 800 || cost != pathCost(t, g, path) {
		t.Fatalf("the path costs %d, want the 800 it takes to walk", cost)
	}
}

func TestFindingTheWayTowardsGoalsFromOneIsNoWayAtAll(t *testing.T) {
	g := newGrid()
	path, cost, ok := pathfinding.FindTowards(utils.Vec2i{}, utils.Vec2i{X: 3}, 3, func(t utils.Vec2i) bool { return t == utils.Vec2i{} }, 10000, g.speed)
	if !ok || path != nil || cost != 0 {
		t.Fatalf("from a goal, found %v costing %d (%v), want nothing to walk", path, cost, ok)
	}
}

func TestFindingTheWayTowardsGoalsThatCantBeReachedGivesUp(t *testing.T) {
	walls := []utils.Vec2i{}
	for y := -30; y <= 30; y++ {
		walls = append(walls, utils.Vec2i{X: 5, Y: y})
	}
	g := newGrid(walls...)
	if _, _, ok := pathfinding.FindTowards(utils.Vec2i{}, utils.Vec2i{X: 8}, 1, func(t utils.Vec2i) bool { return t == utils.Vec2i{X: 7} }, 10000, g.speed); ok {
		t.Fatal("a goal behind a wall across the whole grid was reached")
	}
}

func pathCost(t *testing.T, g grid, path []utils.Vec2i) int {
	t.Helper()
	return cost(t, g, utils.Vec2i{}, path)
}
