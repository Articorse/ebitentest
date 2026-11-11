package tilesystem_test

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/tilesystem"
	"ebittest/utils"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func temperatureAt(t *testing.T, s *testutil.Session, tile utils.Vec2i) float64 {
	t.Helper()
	c, ok := s.Chunks.TemperatureAt(tile)
	if !ok {
		t.Fatalf("%v has no temperature", tile)
	}
	return c.Celsius()
}

// With no wall within 2 tiles of it but its own, which would keep heat as its walls do
func inTheOpen(s *testutil.Session, r int) func(utils.Vec2i) bool {
	return func(middle utils.Vec2i) bool {
		for _, tile := range rect(at(middle, -r-2, -r-2), at(middle, r+2, r+2)) {
			if layers, _ := s.Chunks.TileAt(tile); layers[data.TileLayer_Wall] != data.Tile_None {
				return false
			}
		}
		return true
	}
}

// A sealed 5 by 5 wooden room out in the open, and its walls, at the celsius given, on a planet whose air is times
// Earthlike's, which then turns to the celsius after
func roomThatTurns(t *testing.T, celsius, after float64, times int64) (*testutil.Session, utils.Vec2i) {
	t.Helper()
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Chunks.SetPlanet(constant(t, celsius, times))
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, inTheOpen(s, 3))
	wall(t, s, ring(middle, 3)...)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, 2)))
	roomAt(t, s, middle)
	s.Tick(data.AirStepTicks)
	s.Chunks.SetPlanet(constant(t, after, times))
	return s, middle
}

func TestASealedRoomCoolsThroughItsWallsToTheOutdoors(t *testing.T) {
	s, middle := roomThatTurns(t, 20, 0, 1)
	s.Tick(10 * data.TPS)
	room, wall := temperatureAt(t, s, middle), temperatureAt(t, s, at(middle, 3, 0))
	if !(0 < wall && wall < room && room < 20) {
		t.Fatalf("10 seconds after the outdoors turned 0 °C, the room is %.2f °C and its wall %.2f °C, want the wall between them and the room cooling", room, wall)
	}
	s.Tick(110 * data.TPS)
	if room := temperatureAt(t, s, middle); room < 7 || room > 13 {
		t.Fatalf("two minutes after the outdoors turned 0 °C, the room is %.2f °C, want it about halfway there", room)
	}
	s.Tick(10 * 60 * data.TPS)
	if room := temperatureAt(t, s, middle); room > 1 {
		t.Fatalf("12 minutes after the outdoors turned 0 °C, the room is still %.2f °C", room)
	}
}

func TestAVacuumHasNoTemperatureAndPassesNoHeatOn(t *testing.T) {
	s, middle := roomThatTurns(t, 20, -40, 0)
	if _, ok := s.Chunks.TemperatureAt(middle); ok {
		t.Fatal("a room with no air in it has a temperature")
	}
	s.Tick(30 * data.TPS)
	if wall := temperatureAt(t, s, at(middle, 3, 0)); wall != 20 {
		t.Fatalf("half a minute between a vacuum and a vacuum outdoors, a wall went from 20 °C to %.2f °C", wall)
	}
}

func TestBodiesWarmTheRoomTheyreIn(t *testing.T) {
	s, middle := roomThatTurns(t, 20, 20, 1)
	breather(t, s, middle, data.Body_Giant)
	s.Tick(60 * data.TPS)
	if room := temperatureAt(t, s, middle); room < 21 {
		t.Fatalf("a minute of a giant in a sealed room left it at %.2f °C, want it warmer", room)
	}
}

func TestHeatPassesThroughAWallBetweenTwoRooms(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	s.Chunks.SetPlanet(constant(t, 20, 1))
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, inTheOpen(s, 3))
	wall(t, s, ring(middle, 3)...)
	wall(t, s, at(middle, -2, 0), at(middle, -1, 0), middle, at(middle, 1, 0), at(middle, 2, 0))
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, -1)))
	roof(t, s, rect(at(middle, -2, 1), at(middle, 2, 2)))
	warm, other := at(middle, 0, -1), at(middle, 0, 1)
	breather(t, s, warm, data.Body_Giant)
	s.Tick(2 * 60 * data.TPS)
	shared := temperatureAt(t, s, middle)
	if room := temperatureAt(t, s, other); room <= 20 || shared <= room {
		t.Fatalf("with a giant in the room beside it, the other room is %.2f °C and the wall between them %.2f °C, want the room warmer than the 20 °C outdoors and the wall warmer still", room, shared)
	}
}

func TestOnlyWallsUpToTwoThickKeepHeat(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 5, inTheOpen(s, 5))
	wall(t, s, ring(middle, 3)...)
	wall(t, s, ring(middle, 4)...)
	wall(t, s, ring(middle, 5)...)
	roof(t, s, rect(at(middle, -2, -2), at(middle, 2, 2)))
	s.Tick(data.AirStepTicks)
	for _, c := range []struct {
		tile utils.Vec2i
		keep bool
	}{{at(middle, 3, 0), true}, {at(middle, 4, 0), true}, {at(middle, 5, 0), false}} {
		if _, ok := s.Chunks.TemperatureAt(c.tile); ok != c.keep {
			t.Errorf("%d tiles from the middle of the room, the wall keeps heat %v, want %v", c.tile.X-middle.X, ok, c.keep)
		}
	}
}

func TestHeatIsSaved(t *testing.T) {
	s, middle := roomThatTurns(t, 20, 0, 1)
	s.Tick(7 * data.TPS)
	room, wall := temperatureAt(t, s, middle), temperatureAt(t, s, at(middle, 3, 0))
	s.Commit()
	r := testutil.StartSession(t)
	if got := temperatureAt(t, r, middle); got != room {
		t.Fatalf("after restarting, the room is %.2f °C, want %.2f °C", got, room)
	}
	if got := temperatureAt(t, r, at(middle, 3, 0)); got != wall {
		t.Fatalf("after restarting, the wall is %.2f °C, want %.2f °C", got, wall)
	}
}

func TestTheTemperatureOverlayGoesFromBlueThroughGreenToRed(t *testing.T) {
	cold, mild, hot := drawsystem.TemperatureColor(-1000), drawsystem.TemperatureColor(2000), drawsystem.TemperatureColor(6000)
	if cold.B == 0 || cold.R != 0 || cold.G != 0 {
		t.Errorf("-10 °C is %v, want blue", cold)
	}
	if mild.G == 0 || mild.R != 0 || mild.B != 0 {
		t.Errorf("20 °C is %v, want green", mild)
	}
	if hot.R == 0 || hot.G != 0 || hot.B != 0 {
		t.Errorf("60 °C is %v, want red", hot)
	}
}

func TestWhatsBuiltWeighsWhatItCostsForItsHeat(t *testing.T) {
	read := func(path string, into any) {
		raw, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), path))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, into); err != nil {
			t.Fatal(err)
		}
	}
	type thermal struct {
		Grams int `json:"grams"`
	}
	var blueprints []struct {
		Tile, Feature int
		Costs         []struct{ Grams int }
	}
	var tiles, features []struct {
		Id      int
		Thermal *thermal
	}
	read("assets/blueprints/blueprints.json", &blueprints)
	read("assets/tiles/tiles.json", &tiles)
	read("assets/features/features.json", &features)
	grams := func(defs []struct {
		Id      int
		Thermal *thermal
	}, id int) int {
		for _, d := range defs {
			if d.Id == id && d.Thermal != nil {
				return d.Thermal.Grams
			}
		}
		return -1
	}
	for _, b := range blueprints {
		cost := 0
		for _, c := range b.Costs {
			cost += c.Grams
		}
		got := grams(tiles, b.Tile)
		if b.Feature != 0 {
			got = grams(features, b.Feature)
		}
		if got != cost {
			t.Errorf("what blueprint for tile %d, feature %d builds weighs %d g for its heat, want the %d g it costs", b.Tile, b.Feature, got, cost)
		}
	}
}

func TestOnlyWallsAndDoorsKeepHeatAndAllOfItsNumbersArePositive(t *testing.T) {
	t.Chdir(testutil.RepoRoot())
	write := func(name, json string) string {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	heat := `"thermal": {"specificHeat": 1, "conductance": 1, "grams": 1}`
	for _, c := range []struct {
		name, tile string
		ok         bool
	}{
		{"a wall that keeps heat", `{"id": 1, "imageAsset": "tile_rock", "layer": "wall", "maxHealth": 1, ` + heat + `}`, true},
		{"ground that keeps heat", `{"id": 1, "imageAsset": "tile_dirt", "layer": "ground", "maxHealth": -1, "passable": true, ` + heat + `}`, false},
		{"a wall that weighs nothing", `{"id": 1, "imageAsset": "tile_rock", "layer": "wall", "maxHealth": 1, "thermal": {"specificHeat": 1, "conductance": 1, "grams": 0}}`, false},
		{"a wall that passes no heat on", `{"id": 1, "imageAsset": "tile_rock", "layer": "wall", "maxHealth": 1, "thermal": {"specificHeat": 1, "conductance": 0, "grams": 1}}`, false},
		{"a wall that takes no heat to warm", `{"id": 1, "imageAsset": "tile_rock", "layer": "wall", "maxHealth": 1, "thermal": {"specificHeat": 0, "conductance": 1, "grams": 1}}`, false},
	} {
		if err := tilesystem.NewChunkContainer().GenerateTileAtlasFromJson(write("tiles.json", `[`+c.tile+`]`), testutil.ItemAtlas(t), testutil.Assets(t)); (err == nil) != c.ok {
			t.Errorf("%s: %v, want ok %v", c.name, err, c.ok)
		}
	}
	torch := `{"id": 3, "imageAsset": "torch_placed", "maxHealth": 1, "destroyedInto": 0, ` + heat + `}`
	if err := tilesystem.NewChunkContainer().GenerateFeatureAtlasFromJson(write("features.json", `[`+torch+`]`), testutil.ItemAtlas(t), testutil.Assets(t)); err == nil {
		t.Error("a torch that keeps heat was accepted")
	}
	for _, gas := range []string{
		`{"id": 1, "name": "Oxygen", "condenseCelsius": -183, "item": 3, "specificHeat": 0, "conductance": 1}`,
		`{"id": 1, "name": "Oxygen", "condenseCelsius": -183, "item": 3, "specificHeat": 1, "conductance": -1}`,
	} {
		if _, err := climate.LoadGasesFromJson(write("gases.json", `[`+gas+`]`)); err == nil {
			t.Errorf("the gas %s was accepted", gas)
		}
	}
}
