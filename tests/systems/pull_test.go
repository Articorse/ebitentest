package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

// Between pickup reach and the edge of the pull, as shares of the pull's reach, so that tuning either doesn't break
// these tests
func inPull(share float64) float64 {
	return share * data.ItemPullReachPx
}

// A player whose mattock lies loose at pos, past its pickup delay
func playerWithLooseMattock(t *testing.T, pos utils.Vec2f) (*ecs.ECSContainer, common.EntityId, common.EntityId) {
	t.Helper()
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	testutil.SetPos(t, x, mattock, utils.Vec2f{X: 1000})
	tickItems(x, data.ItemPickupDelayMs)
	testutil.SetPos(t, x, mattock, pos)
	return x, player, mattock
}

func TestNearbyItemsArePulledInFasterEachTick(t *testing.T) {
	// Not a distance that lands exactly on the edge of pickup reach, where rounding would decide
	start := inPull(0.9) + 0.03
	x, player, mattock := playerWithLooseMattock(t, utils.Vec2f{X: start})

	want := start
	for tick := 0; x.Parent(mattock) == -1; tick++ {
		tickItems(x, data.TickMs)
		want -= data.ItemPullStartSpeedPx + float64(tick)*data.ItemPullAccelerationPx
		if want <= data.ItemPickupReachPx {
			break
		}
		if got := testutil.WorldPos(t, x, mattock).X; math.Abs(got-want) > 1e-9 {
			t.Fatalf("after %d ticks of pulling, the mattock is at x %v, want %v", tick+1, got, want)
		}
	}

	if x.Parent(mattock) != player {
		t.Fatal("the mattock wasn't picked up in the tick it was pulled within reach")
	}
}

func TestItemsOutsideThePullAreLeftAlone(t *testing.T) {
	x, _, mattock := playerWithLooseMattock(t, utils.Vec2f{Y: data.ItemPullReachPx + 0.01})
	tickItems(x, 500)

	if pos := testutil.WorldPos(t, x, mattock); pos != (utils.Vec2f{Y: data.ItemPullReachPx + 0.01}) {
		t.Fatalf("an item just outside the pull moved to %v", pos)
	}

	testutil.SetPos(t, x, mattock, utils.Vec2f{Y: data.ItemPullReachPx})
	tickItems(x, data.TickMs)
	if pos := testutil.WorldPos(t, x, mattock); pos.Y >= data.ItemPullReachPx {
		t.Fatal("an item at the edge of the pull wasn't pulled")
	}
}

func TestAPulledItemStopsWhenThePlayerGetsAway(t *testing.T) {
	x, player, mattock := playerWithLooseMattock(t, utils.Vec2f{X: inPull(0.9)})
	tickItems(x, 3*data.TickMs)

	testutil.SetPos(t, x, player, utils.Vec2f{X: -100})
	left := testutil.WorldPos(t, x, mattock)
	tickItems(x, 5*data.TickMs)
	if got := testutil.WorldPos(t, x, mattock); got != left {
		t.Fatalf("the mattock moved from %v to %v after the player got away", left, got)
	}

	// Pulled again from the start, rather than at the speed it had built up
	testutil.SetPos(t, x, player, left.Add(utils.Vec2f{X: inPull(0.6)}))
	tickItems(x, data.TickMs)
	if moved := testutil.WorldPos(t, x, mattock).X - left.X; math.Abs(moved-data.ItemPullStartSpeedPx) > 1e-9 {
		t.Fatalf("pulled again, the mattock moved %v in its first tick, want %v", moved, data.ItemPullStartSpeedPx)
	}
}

func TestItemsThatCantBePickedUpArentPulled(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	inReach := utils.Vec2f{X: inPull(0.6)}
	testutil.SetPos(t, x, mattock, inReach)

	tickItems(x, data.ItemPickupDelayMs-data.TickMs)
	if pos := testutil.WorldPos(t, x, mattock); pos != inReach {
		t.Fatalf("an item was pulled to %v during its pickup delay", pos)
	}

	inv, _ := x.Inventories.Get(player)
	for slot := inv.FirstFreeSlot(); slot != -1; slot = inv.FirstFreeSlot() {
		filler := testutil.AddMovable(t, x, utils.Vec2f{}, false, ecs.NewItemComponent())
		if err := x.PutInInventory(player, slot, filler); err != nil {
			t.Fatal(err)
		}
	}
	tickItems(x, 500)
	if pos := testutil.WorldPos(t, x, mattock); pos != inReach {
		t.Fatalf("an item was pulled to %v towards a full inventory", pos)
	}
}

func TestPulledItemsSlideAlongTerrain(t *testing.T) {
	start := utils.Vec2f{X: -inPull(0.6), Y: inPull(0.5)}
	x, _, mattock := playerWithLooseMattock(t, start)
	x.SetTerrain(blockedLeftOf{openGround: openGround{damaged: map[utils.Vec2i]int{}}, x: start.X + 0.1})

	tickItems(x, 3*data.TickMs)
	pos := testutil.WorldPos(t, x, mattock)
	if pos.X != start.X || pos.Y >= start.Y {
		t.Fatalf("pulled past a wall between it and the player, the mattock is at %v; want it to slide along the wall", pos)
	}
}

// Open ground, but the column of points from x to x+1 is blocked, like a thin wall
type blockedLeftOf struct {
	openGround
	x float64
}

func (b blockedLeftOf) IsBlockedAt(pos utils.Vec2f) bool { return pos.X >= b.x && pos.X < b.x+1 }

func TestItemsArePulledTowardsTheClosestCollector(t *testing.T) {
	x, player, mattock := playerWithLooseMattock(t, utils.Vec2f{})
	testutil.SetPos(t, x, player, utils.Vec2f{X: -inPull(0.8)})
	// Neither is close enough to pick it up in the same tick
	closer := testutil.AddMovable(t, x, utils.Vec2f{X: inPull(0.6)}, false, ecs.NewInventoryComponent(1))
	if closer < player {
		t.Fatal("the test needs the closer collector to come later in entity order, so that order alone can't pick it")
	}

	tickItems(x, data.TickMs)
	if pos := testutil.WorldPos(t, x, mattock); pos.X <= 0 {
		t.Fatalf("the mattock moved to %v, towards the further collector", pos)
	}
}
