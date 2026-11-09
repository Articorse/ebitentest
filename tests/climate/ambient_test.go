package climate_test

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

const seed uint64 = 0x5EED

func dayTicks() uint64 {
	return data.DayTicks + data.DuskTicks + data.NightTicks + data.DawnTicks
}

func yearTicks() uint64 {
	return dayTicks() * data.YearDays
}

// Earthlike's curve without what else moves it
func curveOnly() climate.Planet {
	return climate.Planet{SpawnCelsius: 15, NorthCelsius: -30, SouthCelsius: 35, GradientTiles: 2000}
}

func celsius(p climate.Planet, t utils.Vec2i, tick uint64) float64 {
	return climate.Ambient(p, seed, t, tick).Celsius()
}

func near(got, want, by float64) bool {
	return math.Abs(got-want) <= by
}

func TestItsColderNorthAndWarmerSouthLevellingOffEitherWay(t *testing.T) {
	p := curveOnly()
	if got := celsius(p, utils.Vec2i{}, 0); got != 15 {
		t.Fatalf("at the origin it's %v °C, want 15", got)
	}
	if got := celsius(p, utils.Vec2i{Y: -2000}, 0); !near(got, -30, 0.05*45) {
		t.Fatalf("2000 tiles north it's %v °C, want close to the -30 it levels off at", got)
	}
	if got := celsius(p, utils.Vec2i{Y: -100000}, 0); !near(got, -30, 0.01) {
		t.Fatalf("far north it's %v °C, want -30", got)
	}
	if got := celsius(p, utils.Vec2i{Y: 100000}, 0); !near(got, 35, 0.01) {
		t.Fatalf("far south it's %v °C, want 35", got)
	}
	last := math.Inf(1)
	for y := -3000; y <= 3000; y += 100 {
		got := celsius(p, utils.Vec2i{Y: -y}, 0)
		if got > last {
			t.Fatalf("it's warmer at y %d than 100 tiles south of it", -y)
		}
		last = got
	}
}

func TestTheNoiseVariesItAlongALatitudeWithinItsSwing(t *testing.T) {
	p := curveOnly()
	p.NoiseSwing = 8
	lo, hi := math.Inf(1), math.Inf(-1)
	for x := -20000; x <= 20000; x += 250 {
		got := celsius(p, utils.Vec2i{X: x}, 0)
		if !near(got, 15, 4) {
			t.Fatalf("at x %d it's %v °C, further from 15 than the noise swings", x, got)
		}
		lo, hi = min(lo, got), max(hi, got)
	}
	if hi-lo < 4 {
		t.Fatalf("along the origin's latitude it only ranges from %v to %v °C", lo, hi)
	}
	at := utils.Vec2i{X: 1234, Y: -567}
	if climate.Ambient(p, seed, at, 0) != climate.Ambient(p, seed, at, 0) {
		t.Fatal("the same tile at the same tick has two temperatures")
	}
	if climate.Ambient(p, seed, at, 0) == climate.Ambient(p, seed+1, at, 0) {
		t.Fatal("another world has the same noise")
	}
}

func TestSeasonsSwingMoreFurtherNorth(t *testing.T) {
	p := curveOnly()
	p.SeasonSwing = 20
	midsummer := uint64(float64(yearTicks()) * 3 / 8)
	midwinter := uint64(float64(yearTicks()) * 7 / 8)
	for _, c := range []struct {
		name          string
		y             int
		summer, swing float64
	}{
		{"at the origin", 0, 15, 10},
		{"far north", -100000, -30, 20},
		{"far south", 100000, 35, 0},
	} {
		if got := celsius(p, utils.Vec2i{Y: c.y}, midsummer); !near(got, c.summer+c.swing, 0.05) {
			t.Errorf("%s in midsummer it's %v °C, want %v", c.name, got, c.summer+c.swing)
		}
		if got := celsius(p, utils.Vec2i{Y: c.y}, midwinter); !near(got, c.summer-c.swing, 0.05) {
			t.Errorf("%s in midwinter it's %v °C, want %v", c.name, got, c.summer-c.swing)
		}
	}
	if celsius(p, utils.Vec2i{}, midsummer+yearTicks()) != celsius(p, utils.Vec2i{}, midsummer) {
		t.Fatal("midsummer a year later isn't as warm")
	}
}

func TestItsWarmestAtTheEndOfTheDayAndColdestAtSunrise(t *testing.T) {
	p := curveOnly()
	p.DayNightSwing = 8
	for _, c := range []struct {
		name string
		tick uint64
		want float64
	}{
		{"at sunrise", 0, 11},
		{"at midday", data.DayTicks / 2, 15},
		{"at the end of the day", data.DayTicks, 19},
		{"halfway through the evening and night", data.DayTicks + (dayTicks()-data.DayTicks)/2, 15},
		{"at the next sunrise", dayTicks(), 11},
	} {
		if got := celsius(p, utils.Vec2i{}, c.tick); !near(got, c.want, 0.01) {
			t.Errorf("%s it's %v °C, want %v", c.name, got, c.want)
		}
	}
}

func TestTheYearIsFourSeasonsOfSevenDays(t *testing.T) {
	for _, c := range []struct {
		day      uint64
		season   string
		of, year int
	}{
		{0, "Spring", 1, 1},
		{6, "Spring", 7, 1},
		{7, "Summer", 1, 1},
		{20, "Autumn", 7, 1},
		{27, "Winter", 7, 1},
		{28, "Spring", 1, 2},
		{57, "Spring", 2, 3},
	} {
		season, of, year := climate.Date(c.day*dayTicks() + 1)
		if season != c.season || of != c.of || year != c.year {
			t.Errorf("day %d is %s %d of year %d, want %s %d of year %d", c.day, season, of, year, c.season, c.of, c.year)
		}
	}
}

func TestTheClockRunsFromSunriseAtSixToDuskAtSixAndRoundAgain(t *testing.T) {
	evening := dayTicks() - data.DayTicks
	for _, c := range []struct {
		name         string
		tick         uint64
		hour, minute int
	}{
		{"at sunrise", 0, 6, 0},
		{"halfway through the day", data.DayTicks / 2, 12, 0},
		{"at dusk", data.DayTicks, 18, 0},
		{"halfway from dusk to sunrise", data.DayTicks + evening/2, 0, 0},
		{"a quarter of the way from dusk to sunrise", data.DayTicks + evening/4, 21, 0},
		{"just before sunrise", dayTicks() - 1, 5, 59},
		{"at the next sunrise", dayTicks(), 6, 0},
	} {
		if hour, minute := climate.Clock(c.tick); hour != c.hour || minute != c.minute {
			t.Errorf("%s it's %02d:%02d, want %02d:%02d", c.name, hour, minute, c.hour, c.minute)
		}
	}
}

func TestTheWorldsPlanetIsSaved(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	if s.Chunks.Planet().Id != data.Planet_Earthlike {
		t.Fatalf("a new world is on planet %d, want Earthlike", s.Chunks.Planet().Id)
	}
	moon, err := climate.PlanetOf(testutil.Planets(t), data.Planet_Moon)
	if err != nil {
		t.Fatal(err)
	}
	s.Chunks.SetPlanet(moon)
	s.Commit()
	r := testutil.StartSession(t)
	if r.Chunks.Planet().Id != data.Planet_Moon {
		t.Fatalf("after a restart, the world is on planet %d, want the moon", r.Chunks.Planet().Id)
	}
	if got := r.Chunks.AmbientTemperature(utils.Vec2i{}, 0); got != climate.Ambient(moon, r.Chunks.Seed(), utils.Vec2i{}, 0) {
		t.Fatalf("it's %v outdoors, not the moon's temperature", got)
	}
}

func TestAWorldSavedBeforePlanetsIsOnTheDefaultOne(t *testing.T) {
	p, err := climate.PlanetOf(testutil.Planets(t), data.Planet_None)
	if err != nil || p.Id != data.DefaultPlanet {
		t.Fatalf("a world saved without a planet is on %d (%v), want the default", p.Id, err)
	}
}
