package ecs

import "ebittest/data"

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

// How many tiles a vision of the radius reaches at the darkness
func VisionReach(radius int, darkness float64) float64 {
	night := float64(radius / data.NightVisionDivisor)
	return float64(radius) + (night-float64(radius))*darkness
}
