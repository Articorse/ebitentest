package ecs

import (
	"ebittest/data"
	"ebittest/utils"
	"math"
)

// From 0 in full daylight to 1 at full night, changing evenly over dusk and dawn. Worked out from the tick, so that
// it's saved with it and the same on every run.
func Darkness(tick uint64) float64 {
	t := tick % (data.DayTicks + data.DuskTicks + data.NightTicks + data.DawnTicks)
	switch {
	case t < data.DayTicks:
		return 0
	case t < data.DayTicks+data.DuskTicks:
		return float64(t-data.DayTicks) / data.DuskTicks
	case t < data.DayTicks+data.DuskTicks+data.NightTicks:
		return 1
	default:
		return 1 - float64(t-(data.DayTicks+data.DuskTicks+data.NightTicks))/data.DawnTicks
	}
}

// How brightly the sky lights the tiles open to it
func SkyLight(darkness float64) uint8 {
	return uint8(math.Round(data.DaySkyLight + (data.NightSkyLight-data.DaySkyLight)*darkness))
}

// 0 is pitch black. Only the sky gives light so far, so roofed and unloaded tiles are dark.
func (x *ECSContainer) LightAt(t utils.Vec2i) uint8 {
	if !x.terrain.IsOpenToSky(t) {
		return 0
	}
	return SkyLight(Darkness(x.TickIdx))
}
