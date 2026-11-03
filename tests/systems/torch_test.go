package systems_test

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/systems/animationsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestTheTorchInHandLightsItsHoldersTileAndFlickers(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	inv, _ := x.Inventories.Get(player)
	torches, _ := inv.Slot(4)
	tile := utils.WorldPosToTilePos(testutil.WorldPos(t, x, player))
	lightOnPlayer := func() uint8 {
		t.Helper()
		x.TickIdx = data.DayTicks + data.DuskTicks
		m := &ecs.LightMap{}
		x.LightAround(tile, 0, m)
		return m.At(tile)
	}

	if got := lightOnPlayer(); got != 0 {
		t.Fatalf("with the mattock in hand, the player's tile is lit %d", got)
	}
	press(t, x, player, ecs.InputState{SelectSlot: 5})
	if got := lightOnPlayer(); got == 0 {
		t.Fatal("with a torch in hand, the player's tile is dark")
	}

	spr, _ := x.Sprites.Get(torches)
	frames := []int{}
	for range 4 {
		for range 150 / data.TickMs {
			if err := animationsystem.Tick(x, (*assetmanager.AssetManager)(nil)); err != nil {
				t.Fatal(err)
			}
		}
		frames = append(frames, spr.SubImageIdx())
	}
	if frames[0] == frames[1] && frames[1] == frames[2] {
		t.Fatalf("the torch in hand showed frames %v, want it to flicker", frames)
	}

	press(t, x, player, ecs.InputState{SelectSlot: 1})
	if got := lightOnPlayer(); got != 0 {
		t.Fatalf("with the torch put away, the player's tile is lit %d", got)
	}
}
