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

// Open ground, but blocked from x = wallX on
type wall struct {
	openGround
	wallX float64
}

func (w wall) IsBlockedAt(pos utils.Vec2f) bool { return pos.X >= w.wallX }

// Drops the selected item while aiming at aim, and returns it with the offsets of its sprite during the toss
func dropAndLand(t *testing.T, x *ecs.ECSContainer, player common.EntityId, aim utils.Vec2f) (common.EntityId, []float64) {
	t.Helper()
	inv, _ := x.Inventories.Get(player)
	item, _ := inv.Slot(inv.Selected())
	press(t, x, player, ecs.InputState{DropItem: true, AimPos: aim, HasAim: true})

	heights := []float64{}
	for range data.ItemTossDurationMs/data.TickMs + 1 {
		tickItems(x, data.TickMs)
		spr, _ := x.Sprites.Get(item)
		heights = append(heights, -spr.LocalOffsetPos().Y)
	}
	return item, heights
}

func TestDroppedItemsAreTossedTowardsTheAimInAnArc(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	aim := utils.Vec2f{X: 100, Y: 100}
	item, heights := dropAndLand(t, x, player, aim)

	landed := testutil.WorldPos(t, x, item)
	distance := landed.Length()
	angleOff := math.Abs(math.Atan2(landed.Y, landed.X) - math.Pi/4)
	if distance < data.ItemTossDistancePx*(1-data.ItemTossDistanceVariation)-1e-9 || distance > data.ItemTossDistancePx*(1+data.ItemTossDistanceVariation)+1e-9 {
		t.Errorf("the item landed %v away, want within %v%% of %v", distance, data.ItemTossDistanceVariation*100, data.ItemTossDistancePx)
	}
	if angleOff > data.ItemTossAngleVariationRad+1e-9 {
		t.Errorf("the item landed %v rad off the aim, want at most %v", angleOff, data.ItemTossAngleVariationRad)
	}

	peak, peakAt := 0.0, 0
	for i, h := range heights {
		if h > peak {
			peak, peakAt = h, i
		}
	}
	if peak <= 0 || peak > data.ItemTossHeightPx || peakAt == 0 || peakAt >= len(heights)-2 {
		t.Errorf("the arc went %v up, highest %d ticks into %d, want up to %v and highest in the middle", heights, peakAt, len(heights), data.ItemTossHeightPx)
	}
	if heights[len(heights)-1] != 0 || x.Tosses.HasComponent(item) {
		t.Errorf("after the toss, the item is drawn %v up and still tossed %v; want it down and done", heights[len(heights)-1], x.Tosses.HasComponent(item))
	}
}

func TestTossesVaryButAreTheSameEveryRun(t *testing.T) {
	land := func() [2]utils.Vec2f {
		x, player, _, _, _ := newArmedPlayer(t)
		first, _ := dropAndLand(t, x, player, utils.Vec2f{X: 100})
		press(t, x, player, ecs.InputState{SelectSlot: 2})
		second, _ := dropAndLand(t, x, player, utils.Vec2f{X: 100})
		return [2]utils.Vec2f{testutil.WorldPos(t, x, first), testutil.WorldPos(t, x, second)}
	}

	run := land()
	if math.Abs(run[0].Length()-run[1].Length()) < 1e-6 {
		t.Errorf("two items tossed the same way went the same distance, %v and %v", run[0], run[1])
	}
	if math.Abs(math.Atan2(run[0].Y, run[0].X)-math.Atan2(run[1].Y, run[1].X)) < 1e-6 {
		t.Errorf("two items tossed the same way went in the same direction, %v and %v", run[0], run[1])
	}
	if again := land(); again != run {
		t.Errorf("the same tosses landed at %v, then at %v", run, again)
	}
}

func TestATossStopsAtTerrainButOnlyInItsDirection(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	x.SetTerrain(wall{openGround: openGround{damaged: map[utils.Vec2i]int{}}, wallX: 5})
	item, heights := dropAndLand(t, x, player, utils.Vec2f{X: 100, Y: 100})

	landed := testutil.WorldPos(t, x, item)
	if landed.X >= 5 {
		t.Errorf("the item went into the wall, to %v", landed)
	}
	if landed.Y < data.ItemTossDistancePx*(1-data.ItemTossDistanceVariation)/math.Sqrt2-1 {
		t.Errorf("the item landed at %v, so it stopped moving down along the wall too", landed)
	}
	if heights[len(heights)-1] != 0 || x.Tosses.HasComponent(item) {
		t.Error("the toss didn't finish its arc after meeting the wall")
	}
}

func TestWithoutAnAimItemsAreDroppedAtTheFeet(t *testing.T) {
	x, player, mattock, _, _ := newArmedPlayer(t)
	press(t, x, player, ecs.InputState{DropItem: true})
	tickItems(x, data.ItemTossDurationMs)

	if x.Tosses.HasComponent(mattock) || testutil.WorldPos(t, x, mattock) != (utils.Vec2f{}) {
		t.Fatalf("dropped without an aim, the mattock is at %v, tossed %v; want it at the feet", testutil.WorldPos(t, x, mattock), x.Tosses.HasComponent(mattock))
	}
}
