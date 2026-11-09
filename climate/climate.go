package climate

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Added to the world seed, apart from worldgen's, so that the climate's noise is its own
const climateSeed uint64 = 1000

// How warm a planet is outdoors. Swings are from the coldest to the warmest, half either side.
type Planet struct {
	Id   data.PlanetEnum
	Name string
	// The mean at the latitude of the world's origin, and what it levels off at far north and far south
	SpawnCelsius, NorthCelsius, SouthCelsius float64
	// About how far from the origin it's levelled off
	GradientTiles float64
	NoiseSwing    float64
	// At the origin's latitude; it's nothing far south and twice as much far north
	SeasonSwing   float64
	DayNightSwing float64
	// Per tile, the same everywhere and always
	Atmosphere Air
}

type planetDto struct {
	Id            uint16  `json:"id"`
	Name          string  `json:"name"`
	SpawnCelsius  float64 `json:"spawnCelsius"`
	NorthCelsius  float64 `json:"northCelsius"`
	SouthCelsius  float64 `json:"southCelsius"`
	GradientTiles float64 `json:"gradientTiles"`
	NoiseSwing    float64 `json:"noiseSwing"`
	SeasonSwing   float64 `json:"seasonSwing"`
	DayNightSwing float64 `json:"dayNightSwing"`
	Atmosphere    []struct {
		Gas   uint16 `json:"gas"`
		Grams int64  `json:"grams"`
	} `json:"atmosphere"`
}

func LoadPlanetsFromJson(path string, gases map[data.GasEnum]Gas) (map[data.PlanetEnum]Planet, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading planets json file: %w", err)
	}
	var dtos []planetDto
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling planets json: %w", err)
	}
	planets := map[data.PlanetEnum]Planet{}
	for _, d := range dtos {
		id := data.PlanetEnum(d.Id)
		switch _, taken := planets[id]; {
		case id == data.Planet_None:
			return nil, fmt.Errorf("planet %q has id 0, which means none", d.Name)
		case taken:
			return nil, fmt.Errorf("planet id %d is defined twice", d.Id)
		case d.GradientTiles <= 0:
			return nil, fmt.Errorf("planet %d has a gradient of %v tiles, want it positive", d.Id, d.GradientTiles)
		}
		p := Planet{
			Id: id, Name: d.Name,
			SpawnCelsius: d.SpawnCelsius, NorthCelsius: d.NorthCelsius, SouthCelsius: d.SouthCelsius,
			GradientTiles: d.GradientTiles, NoiseSwing: d.NoiseSwing, SeasonSwing: d.SeasonSwing, DayNightSwing: d.DayNightSwing,
		}
		for _, a := range d.Atmosphere {
			gas := data.GasEnum(a.Gas)
			switch _, known := gases[gas]; {
			case !known:
				return nil, fmt.Errorf("planet %d's atmosphere has gas %d, which isn't defined", d.Id, a.Gas)
			case a.Grams < 0:
				return nil, fmt.Errorf("planet %d's atmosphere has %d g of gas %d, want it positive", d.Id, a.Grams, a.Gas)
			}
			p.Atmosphere[gas] += a.Grams * data.MilligramsPerGram
		}
		planets[id] = p
	}
	if _, ok := planets[data.DefaultPlanet]; !ok {
		return nil, fmt.Errorf("the default planet %d isn't defined", data.DefaultPlanet)
	}
	return planets, nil
}

// The planet a world saved with id is on, where a world saved before worlds had planets is on the default one
func PlanetOf(planets map[data.PlanetEnum]Planet, id data.PlanetEnum) (Planet, error) {
	if id == data.Planet_None {
		id = data.DefaultPlanet
	}
	p, ok := planets[id]
	if !ok {
		return Planet{}, fmt.Errorf("planet %d isn't defined", id)
	}
	return p, nil
}

// In hundredths of a degree, so that what goes by it is worked out the same on every machine, whatever the last bits of
// the float maths come to
type Centicelsius int32

func (c Centicelsius) Celsius() float64 {
	return float64(c) / 100
}

// The temperature outdoors on the tile at the tick, which depends only on them, the seed and the planet, so it's the
// same on every run and needs nothing saved
func Ambient(p Planet, seed uint64, t utils.Vec2i, tick uint64) Centicelsius {
	half := p.GradientTiles / 2
	// tanh(2) is 0.96, so it's all but levelled off GradientTiles from the origin
	celsius := p.SpawnCelsius
	if t.Y < 0 {
		celsius += (p.NorthCelsius - p.SpawnCelsius) * math.Tanh(float64(-t.Y)/half)
	} else {
		celsius += (p.SouthCelsius - p.SpawnCelsius) * math.Tanh(float64(t.Y)/half)
	}
	celsius += p.NoiseSwing / 2 * noise.OpenSimplex2(int64(seed+climateSeed), float64(t.X)/data.ClimateNoiseScaleTiles, float64(t.Y)/data.ClimateNoiseScaleTiles)

	northness := (1 + math.Tanh(float64(-t.Y)/half)) / 2
	celsius += p.SeasonSwing * northness * math.Cos(2*math.Pi*(YearFraction(tick)-data.MidsummerYearFraction))
	celsius += p.DayNightSwing / 2 * dayWarmth(tick)
	return Centicelsius(math.Round(celsius * 100))
}

func dayTicks() uint64 {
	return data.DayTicks + data.DuskTicks + data.NightTicks + data.DawnTicks
}

// How far through the year the tick is, from 0 at the start of spring
func YearFraction(tick uint64) float64 {
	year := dayTicks() * data.YearDays
	return float64(tick%year) / float64(year)
}

// -1 at sunrise, warming to 1 by the end of the day, then cooling back down over dusk, the night and dawn
func dayWarmth(tick uint64) float64 {
	t := tick % dayTicks()
	if t < data.DayTicks {
		return -1 + 2*float64(t)/data.DayTicks
	}
	return 1 - 2*float64(t-data.DayTicks)/float64(dayTicks()-data.DayTicks)
}

var seasonNames = [4]string{"Spring", "Summer", "Autumn", "Winter"}

// The season, the day of it and the year, from 1
func Date(tick uint64) (season string, day, year int) {
	days := int(tick / dayTicks())
	ofYear := days % data.YearDays
	return seasonNames[ofYear/data.SeasonDays], ofYear%data.SeasonDays + 1, days/data.YearDays + 1
}

// The time on a 24 hour clock, with the day from sunrise at 06:00 to dusk at 18:00 and dusk, the night and dawn
// sharing the rest, so that each keeps its share of the cycle whatever their lengths
func Clock(tick uint64) (hour, minute int) {
	const dayMinutes = 24 * 60
	t := tick % dayTicks()
	var minutes uint64
	if t < data.DayTicks {
		minutes = 6*60 + t*dayMinutes/2/data.DayTicks
	} else {
		minutes = 18*60 + (t-data.DayTicks)*dayMinutes/2/(dayTicks()-data.DayTicks)
	}
	minutes %= dayMinutes
	return int(minutes / 60), int(minutes % 60)
}
