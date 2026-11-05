package pathfinding_test

import (
	"ebittest/pathfinding"
	"ebittest/utils"
	"testing"
)

// The grid's walkable square
var area = [2]utils.Vec2i{{X: -30, Y: -30}, {X: 30, Y: 30}}

func TestTheNearestGoalIsTheClosestToWalkToNotAsTheCrowFlies(t *testing.T) {
	// A wall at x = 3 from y = -10 to 10 hides the goal 4 tiles to the right behind it
	walls := []utils.Vec2i{}
	for y := -10; y <= 10; y++ {
		walls = append(walls, utils.Vec2i{X: 3, Y: y})
	}
	g := newGrid(walls...)
	behind, open := utils.Vec2i{X: 4}, utils.Vec2i{X: -7}
	isGoal := func(t utils.Vec2i) bool { return t == behind || t == open }

	path, ok := pathfinding.FindNearest(utils.Vec2i{}, area[0], area[1], isGoal, 10000, g.speed)
	if !ok || len(path) == 0 || path[len(path)-1] != open {
		t.Fatalf("found %v (%v), want the open goal 7 tiles away rather than the one 4 tiles away behind the wall", path, ok)
	}
	if want := cost(t, g, utils.Vec2i{}, path); want != 700 {
		t.Fatalf("the path costs %d, want the straight 700", want)
	}
}

func TestNoNearestGoalIsFoundWhereNoneCanBeReached(t *testing.T) {
	g := newGrid(utils.Vec2i{X: 1}, utils.Vec2i{X: -1}, utils.Vec2i{Y: 1}, utils.Vec2i{Y: -1},
		utils.Vec2i{X: 1, Y: 1}, utils.Vec2i{X: -1, Y: 1}, utils.Vec2i{X: 1, Y: -1}, utils.Vec2i{X: -1, Y: -1})
	atFive := func(t utils.Vec2i) bool { return t == utils.Vec2i{X: 5} }
	if _, ok := pathfinding.FindNearest(utils.Vec2i{}, area[0], area[1], atFive, 10000, g.speed); ok {
		t.Fatal("a goal was reached from a walled in start")
	}
	atStart := func(t utils.Vec2i) bool { return t == utils.Vec2i{} }
	if path, ok := pathfinding.FindNearest(utils.Vec2i{}, area[0], area[1], atStart, 10000, g.speed); !ok || path != nil {
		t.Fatal("standing on a goal didn't count as having reached it")
	}
}

func TestTheNearestGoalSearchGivesUpAfterItsLimit(t *testing.T) {
	g := newGrid()
	far := utils.Vec2i{X: 25}
	if _, ok := pathfinding.FindNearest(utils.Vec2i{}, area[0], area[1], func(t utils.Vec2i) bool { return t == far }, 100, g.speed); ok {
		t.Fatal("a goal 25 tiles away was found expanding 100 tiles")
	}
}

func TestTheNearestGoalSearchStaysInItsArea(t *testing.T) {
	g := newGrid()
	outside := utils.Vec2i{X: 6}
	small := [2]utils.Vec2i{{X: -5, Y: -5}, {X: 5, Y: 5}}
	if _, ok := pathfinding.FindNearest(utils.Vec2i{}, small[0], small[1], func(t utils.Vec2i) bool { return t == outside }, 10000, g.speed); ok {
		t.Fatal("a goal outside the area searched was found")
	}
	if _, ok := pathfinding.FindNearest(outside, small[0], small[1], func(t utils.Vec2i) bool { return t == utils.Vec2i{} }, 10000, g.speed); ok {
		t.Fatal("a search starting outside its area found a goal")
	}
}
