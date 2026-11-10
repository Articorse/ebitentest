package tilesystem_test

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"testing"
)

// A 7×7 square of open ground, far from the colonist, with its middle returned
func doorYard(t *testing.T, s *testutil.Session, colonist common.EntityId) utils.Vec2i {
	t.Helper()
	corner := storageSpot(t, s, colonist, 12, 7, 7)
	return utils.Vec2i{X: corner.X + 3, Y: corner.Y + 3}
}

func at(t utils.Vec2i, dx, dy int) utils.Vec2i {
	return utils.Vec2i{X: t.X + dx, Y: t.Y + dy}
}

func wall(t *testing.T, s *testutil.Session, tiles ...utils.Vec2i) {
	t.Helper()
	for _, tile := range tiles {
		if !s.Chunks.PlaceTile(tile, data.Tile_WoodWall) {
			t.Fatalf("couldn't put a wall on %v", tile)
		}
	}
}

// Placed, supplied and built for the faction
func buildDoor(t *testing.T, s *testutil.Session, tile utils.Vec2i, faction data.FactionEnum) {
	t.Helper()
	if !s.Chunks.PlaceSite(tile, data.Blueprint_WoodDoor) || !s.Chunks.SetSiteFaction(tile, faction) {
		t.Fatalf("couldn't place a door on %v", tile)
	}
	s.Chunks.DeliverToSite(tile, data.Item_Wood, 5*kg)
	if _, built := s.Chunks.BuildSite(tile, 1_000_000, true); !built {
		t.Fatalf("the door on %v wasn't built", tile)
	}
}

func door(t *testing.T, s *testutil.Session, tile utils.Vec2i) tilesystem.Door {
	t.Helper()
	d, ok := s.Chunks.DoorAt(tile)
	if !ok {
		t.Fatalf("there's no door on %v", tile)
	}
	return d
}

func TestADoorFacesAlongWhatItJoinsOnto(t *testing.T) {
	for _, c := range []struct {
		name     string
		walls    []utils.Vec2i
		vertical bool
	}{
		{"on its own", nil, false},
		{"between walls to either side", []utils.Vec2i{{X: -1}, {X: 1}}, false},
		{"with a wall to one side", []utils.Vec2i{{X: 1}}, false},
		{"between walls above and below", []utils.Vec2i{{Y: -1}, {Y: 1}}, true},
		{"with a wall below", []utils.Vec2i{{Y: 1}}, true},
		{"with walls all round", []utils.Vec2i{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _, colonist := colonistAtWork(t)
			middle := doorYard(t, s, colonist)
			for _, d := range c.walls {
				wall(t, s, at(middle, d.X, d.Y))
			}
			buildDoor(t, s, middle, data.Faction_Colony)
			if d := door(t, s, middle); d.Vertical != c.vertical || d.Open {
				t.Fatalf("the door is %+v, want vertical %v and closed", d, c.vertical)
			}
		})
	}
}

func TestADoorTurnsWhenWhatItJoinsOntoChanges(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	buildDoor(t, s, middle, data.Faction_Colony)

	above, left := at(middle, 0, -1), at(middle, -1, 0)
	s.Chunks.PlaceSite(above, data.Blueprint_WoodWall)
	if !door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn to a wall blueprint placed above it")
	}
	s.Chunks.RemoveSite(above)
	if door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn back once the blueprint was taken away")
	}
	s.Chunks.PlaceSite(above, data.Blueprint_WoodDoor)
	if !door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn to a door blueprint placed above it")
	}
	s.Chunks.RemoveSite(above)
	buildDoor(t, s, at(middle, 0, 1), data.Faction_Colony)
	if !door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn to a door below it")
	}
	wall(t, s, left)
	if door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn to a wall built beside it")
	}
	if destroyed, _ := s.Chunks.DamageTile(left, 1_000_000); !destroyed {
		t.Fatal("the wall wasn't destroyed")
	}
	if !door(t, s, middle).Vertical {
		t.Fatal("the door didn't turn back once the wall beside it was gone")
	}
}

func TestAClosedDoorBlocksSightAndBodiesButNotPaths(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, at(middle, -1, 0), at(middle, 1, 0))
	buildDoor(t, s, middle, data.Faction_Colony)
	center := utils.TilePosToWorldPos(middle)

	opaque := make([]bool, 1)
	s.Chunks.TileFlags(middle, 1, 1, opaque, nil)
	if !s.Chunks.IsOpaqueTile(middle) || !opaque[0] || !s.Chunks.IsBlockedAt(center) || s.Chunks.IsBlockedAt(center.Add(utils.Vec2f{Y: 4})) {
		t.Fatal("a closed door doesn't block sight, or its middle row, or blocks more than its pixels")
	}
	if s.Chunks.WalkSpeedPercent(middle) <= 0 {
		t.Fatal("paths don't go through a closed door")
	}

	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, 1)))
	s.Chunks.TickDoors(s.ECS)
	s.Chunks.TileFlags(middle, 1, 1, opaque, nil)
	if !door(t, s, middle).Open || s.Chunks.IsOpaqueTile(middle) || opaque[0] || s.Chunks.IsBlockedAt(center) {
		t.Fatal("an open door still blocks sight or bodies")
	}
}

func TestADoorOpensForItsOwnFactionWithinATile(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, at(middle, -1, 0), at(middle, 1, 0))
	buildDoor(t, s, middle, data.Faction_Colony)
	slime, err := templates.SpawnEvilSlime(s.ECS, utils.TilePosToWorldPos(at(middle, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, 2)))
	s.Chunks.TickDoors(s.ECS)
	if door(t, s, middle).Open {
		t.Fatal("the door opened for a monster next to it, or for a colonist two tiles away")
	}

	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, -1, -1)))
	s.Chunks.TickDoors(s.ECS)
	if !door(t, s, middle).Open {
		t.Fatal("the door didn't open for a colonist diagonally next to it")
	}

	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, 3)))
	testutil.SetPos(t, s.ECS, slime, utils.TilePosToWorldPos(middle))
	s.Chunks.TickDoors(s.ECS)
	if !door(t, s, middle).Open {
		t.Fatal("the door closed on a monster in its way")
	}
	testutil.SetPos(t, s.ECS, slime, utils.TilePosToWorldPos(at(middle, 0, 3)))
	s.Chunks.TickDoors(s.ECS)
	if door(t, s, middle).Open {
		t.Fatal("the door didn't close once nothing was in its way and no one of its faction was near")
	}
}

func TestADoorWithoutAFactionNeverOpens(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, at(middle, -1, 0), at(middle, 1, 0))
	buildDoor(t, s, middle, data.Faction_None)
	testutil.AddMovable(t, s.ECS, utils.TilePosToWorldPos(at(middle, 0, 1)), false, ecs.NewFactionComponent(data.Faction_None))
	s.Chunks.TickDoors(s.ECS)
	if door(t, s, middle).Open {
		t.Fatal("the door without a faction opened for something without one")
	}
}

func TestDoorsInARowOpenTogether(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, at(middle, -2, 0), at(middle, 2, 0))
	row := []utils.Vec2i{at(middle, -1, 0), middle, at(middle, 1, 0)}
	for _, tile := range row {
		buildDoor(t, s, tile, data.Faction_Colony)
	}
	// Below the middle of the row, facing the other way, and so not in it
	below := at(middle, 0, 1)
	buildDoor(t, s, below, data.Faction_Colony)

	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(row[0], -1, -1)))
	s.Chunks.TickDoors(s.ECS)
	for _, tile := range row {
		if !door(t, s, tile).Open {
			t.Fatalf("the door on %v didn't open with the rest of its row", tile)
		}
	}
	if d := door(t, s, below); d.Open || !d.Vertical {
		t.Fatalf("the door facing another way is %+v, want it closed and vertical", d)
	}

	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(below, 0, 1)))
	s.Chunks.TickDoors(s.ECS)
	if !door(t, s, below).Open || door(t, s, middle).Open {
		t.Fatal("the door below the row didn't open on its own")
	}
}

func TestADoorKeepsItsHealthOpenAndClosedAndDropsItsWood(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	buildDoor(t, s, middle, data.Faction_Colony)
	if health, ok := s.Chunks.FeatureHealth(middle); !ok || health != 50 {
		t.Fatalf("a door has %d health, want 50", health)
	}
	s.Chunks.DamageFeature(middle, 10)
	testutil.SetPos(t, s.ECS, colonist, utils.TilePosToWorldPos(at(middle, 0, 1)))
	s.Chunks.TickDoors(s.ECS)
	if health, _ := s.Chunks.FeatureHealth(middle); !door(t, s, middle).Open || health != 40 {
		t.Fatalf("an opened door has %d health, want the 40 it had", health)
	}

	destroyed, drops := s.Chunks.DamageFeature(middle, 1_000)
	if !destroyed {
		t.Fatal("the door wasn't destroyed")
	}
	s.ECS.LayDownDrops(drops, middle)
	if kind, n := onGround(s, middle); kind != data.Item_Wood || n != 5*kg {
		t.Fatalf("a destroyed door left %d mg of %v, want its 5 kg of wood", n, kind)
	}
	if _, ok := s.Chunks.DoorAt(middle); ok {
		t.Fatal("the door is still there")
	}
	s.Chunks.PlaceSite(middle, data.Blueprint_WoodDoor)
	s.Chunks.DeliverToSite(middle, data.Item_Wood, 5*kg)
	s.Chunks.BuildSite(middle, 1_000_000, true)
	if d := door(t, s, middle); d.Faction != data.Faction_None {
		t.Fatalf("a door built for no one in place of the colony's is the faction %v's", d.Faction)
	}
}

func TestADoorGoesInPlaceOfAWallBlueprintOnly(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	blueprint, supplied, ground := middle, at(middle, 1, 0), at(middle, -2, 0)
	s.Chunks.PlaceSite(blueprint, data.Blueprint_WoodWall)
	s.Chunks.PlaceSite(supplied, data.Blueprint_WoodWall)
	s.Chunks.DeliverToSite(supplied, data.Item_Wood, 1*kg)

	if !s.Chunks.CanPlaceBlueprint(ground, data.Blueprint_WoodDoor) || s.Chunks.CanPlaceBlueprint(supplied, data.Blueprint_WoodDoor) {
		t.Fatal("a door can't go on open ground, or can on a wall with wood delivered to it")
	}
	if !s.Chunks.PlaceSite(blueprint, data.Blueprint_WoodDoor) {
		t.Fatal("a door wasn't placed in place of the wall blueprint")
	}
	if site, _ := s.Chunks.Site(blueprint); site.Blueprint != data.Blueprint_WoodDoor {
		t.Fatalf("the site is %v, want the door", site.Blueprint)
	}
	if s.Chunks.CanPlaceBlueprint(blueprint, data.Blueprint_WoodWall) || s.Chunks.CanPlaceBlueprint(blueprint, data.Blueprint_WoodDoor) {
		t.Fatal("something can go in place of the door blueprint")
	}
}

func TestADoorPlacedForAFactionIsItsAndSaved(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	s.Chunks.PlaceSite(middle, data.Blueprint_WoodWall)
	handleInput(t, s, colonist, ecs.InputState{PlaceBlueprint: ecs.BlueprintPlacement{Kind: data.Blueprint_WoodDoor, Tiles: []utils.Vec2i{middle}}})
	site, ok := s.Chunks.Site(middle)
	if !ok || site.Blueprint != data.Blueprint_WoodDoor || site.Faction != data.Faction_Colony || site.Order.Id != s.ECS.LastOrder {
		t.Fatalf("the site is %+v, want the colony's door in a new order", site)
	}
	s.Chunks.DeliverToSite(middle, data.Item_Wood, 5*kg)
	s.Chunks.BuildSite(middle, 1_000_000, true)
	s.Commit()

	r := testutil.StartSession(t)
	if d, ok := r.Chunks.DoorAt(middle); !ok || d.Faction != data.Faction_Colony {
		t.Fatalf("after restarting the door is %+v (%v), want the colony's", d, ok)
	}
}

func TestADoorBlueprintLooksLikeTheDoorItWillBe(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, at(middle, 0, -1))
	s.Chunks.PlaceSite(middle, data.Blueprint_WoodDoor)
	if sprite := siteImage(t, s, middle); !sprite.Ghost || sprite.ImageAsset != "wood_door" || sprite.Frame != 2 {
		t.Fatalf("the door blueprint is drawn as %+v, want the ghost of a closed vertical door", sprite)
	}
}

func TestADoorBeingBuiltShowsItsStages(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	s.Chunks.PlaceSite(middle, data.Blueprint_WoodDoor)
	s.Chunks.DeliverToSite(middle, data.Item_Wood, 5*kg)
	for {
		if sprite := siteImage(t, s, middle); sprite.Ghost || sprite.ImageAsset != "tile_planks" || sprite.Frame < 1 || sprite.Frame > 3 {
			t.Fatalf("the door being built is drawn as %+v, want a stage of the planks", sprite)
		}
		if worked, _ := s.Chunks.BuildSite(middle, 1, false); !worked {
			return
		}
	}
}

func TestColonistsGoThroughTheirDoors(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	for d := -3; d <= 3; d++ {
		for _, tile := range []utils.Vec2i{at(middle, d, -3), at(middle, d, 3), at(middle, -3, d), at(middle, 3, d)} {
			if _, ok := s.Chunks.TileAt(tile); ok && tile != at(middle, 0, 3) && tile != at(middle, 0, -3) && !isWallAt(s, tile) {
				wall(t, s, tile)
			}
		}
	}
	buildDoor(t, s, at(middle, 0, 3), data.Faction_Colony)
	buildDoor(t, s, at(middle, 0, -3), data.Faction_Colony)
	inside := middle
	s.Chunks.PlaceSite(inside, data.Blueprint_WoodWall)

	// Bumping into the doors as in the game, so that they have to open to be got through
	liveColliding(t, s, 60*data.TPS, []common.EntityId{colonist})
	if !isBuilt(s, inside)() {
		t.Fatalf("the colonist didn't build the wall behind the doors, it's %v on %v", colonistState(s, colonist), colonistTile(t, s, colonist))
	}
}

func isWallAt(s *testutil.Session, tile utils.Vec2i) bool {
	return currentTile(s, tile) == data.Tile_WoodWall
}
