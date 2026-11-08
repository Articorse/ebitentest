package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"os"
	"path/filepath"
	"testing"
)

func placeTorch(t *testing.T, s *testutil.Session) utils.Vec2i {
	t.Helper()
	return placeTorchOn(t, s, bareGrass(t, s))
}

func placeTorchOn(t *testing.T, s *testutil.Session, tile utils.Vec2i) utils.Vec2i {
	t.Helper()
	if s.Chunks.PlaceFeature(tile, data.Feature_None) {
		t.Fatal("nothing was placed as a feature")
	}
	if !s.Chunks.CanPlaceFeature(tile) || !s.Chunks.PlaceFeature(tile, data.Feature_Torch) {
		t.Fatal("a torch wasn't placed on bare grass")
	}
	return tile
}

func TestPlacedTorchesAreWalkedThroughAndAnyHitBreaksThemIntoATorch(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := placeTorch(t, s)
	if s.Chunks.CanPlaceFeature(tile) || s.Chunks.PlaceFeature(tile, data.Feature_Torch) || s.Chunks.CanPlaceTile(tile) {
		t.Fatal("something can be placed on a torch")
	}

	s.Commit()
	r := testutil.StartSession(t)
	if f, ok := r.Chunks.FeatureAt(tile); !ok || f != data.Feature_Torch {
		t.Fatalf("after restarting, the tile has feature %v (%v), want the torch", f, ok)
	}

	center := utils.TilePosToWorldPos(tile)
	if r.Chunks.IsBlockedAt(center) {
		t.Fatal("the torch blocks")
	}
	mob := addCircle(t, r.ECS, center, 3, func(shape shapes.Shape) ecs.Component {
		return ecs.NewPhysicsColliderComponent(ecs.Layer_Player, ecs.Layer_Wall|ecs.Layer_Feature, ecs.Collider_Mob, shape)
	})
	resolveTerrain(t, r)
	if got := testutil.WorldPos(t, r.ECS, mob); got != center {
		t.Fatalf("a mob standing on the torch was pushed to %v", got)
	}

	bullet := addCircle(t, r.ECS, center, 1, func(shape shapes.Shape) ecs.Component {
		return ecs.NewHurtboxColliderComponent(ecs.Layer_FriendlyProjectile, ecs.Layer_Wall|ecs.Layer_Feature, shape)
	}, func(shapes.Shape) ecs.Component { return ecs.NewContactDamageComponent(-1, 0, 1).WithDieOnContact() })
	touchTerrain(t, r)

	if !r.ECS.HasEntity(bullet) {
		t.Fatal("a projectile died on the torch, as if it were a wall")
	}
	if _, ok := r.Chunks.FeatureAt(tile); ok {
		t.Fatal("a hit of 1 didn't break the torch")
	}
	if kind, n := onGround(r, tile); kind != data.Item_Torch || n != 1 {
		t.Fatalf("the broken torch left %d of %v on its tile, want a torch", n, kind)
	}
	if !r.Chunks.CanPlaceFeature(tile) {
		t.Fatal("a torch can't be placed where one was broken")
	}
}

func TestTorchesAreOnlyLightSourcesWhereTheirLightReaches(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	// The torch's light, as the feature atlas has it, and its range
	var level uint8
	var reach int
	// Over the tiles from fromX to toX tiles across from the torch, whichever way round, and 5 tiles down
	litBy := func(torch utils.Vec2i, fromX, toX int) bool {
		found := false
		min := utils.Vec2i{X: torch.X + min(fromX, toX), Y: torch.Y}
		max := utils.Vec2i{X: torch.X + max(fromX, toX), Y: torch.Y + 5}
		s.Chunks.LightSources(min, max, func(t utils.Vec2i, l uint8, rangeTiles int) {
			if t == torch {
				found, level, reach = true, l, rangeTiles
			}
		})
		return found
	}

	// On the edges of chunks, so that their light reaches into the next ones on either side
	for _, outwards := range []int{1, -1} {
		torch := findTile(t, s, func(tile utils.Vec2i) bool {
			_, hasFeature := s.Chunks.FeatureAt(tile)
			next := utils.Vec2i{X: tile.X + outwards, Y: tile.Y}
			return currentTile(s, tile) == data.Tile_Grass && !hasFeature &&
				utils.WorldPosToChunkGridPos(utils.TilePosToWorldPos(tile)) != utils.WorldPosToChunkGridPos(utils.TilePosToWorldPos(next)) &&
				s.Chunks.IsActiveWorldPos(utils.TilePosToWorldPos(next))
		})
		if litBy(torch, 0, 0) {
			t.Fatalf("before the torch at %v is placed, it gives light", torch)
		}
		placeTorchOn(t, s, torch)

		if !litBy(torch, 0, 0) || level <= data.NightSkyLight || reach < 2 {
			t.Fatalf("the torch at %v gives light %d up to %d tiles away, want more than the night sky's over 2 tiles or more", torch, level, reach)
		}
		if !litBy(torch, reach*outwards, (reach+5)*outwards) {
			t.Fatalf("the torch at %v doesn't light the next chunk %d tiles away", torch, reach)
		}
		if litBy(torch, -(reach+1)*outwards, -(reach+6)*outwards) {
			t.Fatalf("the torch at %v lights its own chunk %d tiles away", torch, reach+1)
		}

		s.ECS.TickIdx = data.DayTicks + data.DuskTicks
		m := &ecs.LightMap{}
		s.ECS.LightAround(torch, 0, m)
		if got := m.At(torch); got != level {
			t.Fatalf("at night, the torch's tile is lit %d, want %d", got, level)
		}

		s.ECS.DamageFeature(torch, 1)
		if litBy(torch, 0, 0) {
			t.Fatalf("after breaking the torch at %v, it still gives light", torch)
		}
	}
}

func TestPlacedTorchesFlickerButAreRememberedStill(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tile := placeTorch(t, s)
	center := utils.TilePosToWorldPos(tile)

	for ms, want := range map[uint64]int{0: 0, 149: 0, 150: 1, 300: 2, 450: 1, 599: 1, 600: 0, 750: 1} {
		sprites := s.Chunks.FeatureSprites([2]utils.Vec2f{center, center}, ms)
		if len(sprites) != 1 || sprites[0].ImageAsset != common.AssetSheetTorchPlaced || sprites[0].Frame != want {
			t.Errorf("at %dms the torch is drawn as %+v, want frame %d", ms, sprites, want)
		}
	}

	see(s, center)
	remembered := s.Chunks.RememberedFeatureSprites([2]utils.Vec2f{center, center})
	if len(remembered) != 1 || remembered[0].Frame != 0 {
		t.Fatalf("the remembered torch is drawn as %+v, want its first frame", remembered)
	}
}

func TestTheFeatureAtlasRefusesBrokenLightsAndAnimations(t *testing.T) {
	items, err := ecs.LoadItemAtlasFromJson(filepath.Join(testutil.RepoRoot(), testutil.ItemAtlasPath))
	if err != nil {
		t.Fatal(err)
	}
	load := func(feature string) error {
		path := filepath.Join(t.TempDir(), "features.json")
		if err := os.WriteFile(path, []byte(`[`+feature+`]`), 0o644); err != nil {
			t.Fatal(err)
		}
		return tilesystem.NewChunkContainer().GenerateFeatureAtlasFromJson(path, items, testutil.Assets(t))
	}

	if err := load(`{"id": 3, "imageAsset": "torch_placed", "maxHealth": 1, "destroyedInto": 0, "light": {"level": 150, "rangeTiles": 10},
		"animation": [{"frame": 0, "durationMs": 150}, {"frame": 2, "durationMs": 150}]}`); err != nil {
		t.Fatalf("a torch that leaves nothing behind was refused: %v", err)
	}
	for _, c := range []struct {
		name    string
		feature string
	}{
		{"no light", `{"id": 3, "imageAsset": "torch_placed", "light": {"level": 0, "rangeTiles": 10}}`},
		{"light reaching nowhere", `{"id": 3, "imageAsset": "torch_placed", "light": {"level": 150, "rangeTiles": 0}}`},
		{"a frame that doesn't last", `{"id": 3, "imageAsset": "torch_placed", "animation": [{"frame": 0, "durationMs": 0}]}`},
		{"a negative frame", `{"id": 3, "imageAsset": "torch_placed", "animation": [{"frame": -1, "durationMs": 150}]}`},
		{"a frame past its image's last", `{"id": 3, "imageAsset": "torch_placed", "animation": [{"frame": 3, "durationMs": 150}]}`},
		{"an image that isn't in the asset list", `{"id": 3, "imageAsset": "nothing"}`},
		{"destruction into a missing feature", `{"id": 3, "imageAsset": "torch_placed", "maxHealth": 1, "destroyedInto": 7}`},
	} {
		if err := load(c.feature); err == nil {
			t.Errorf("a feature with %s was accepted", c.name)
		}
	}
}
