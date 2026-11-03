package data

import "image/color"

// The day starts at tick 0 and runs day, dusk, night, dawn
const (
	DayTicks   = 10 * TPS
	DuskTicks  = 5 * TPS
	NightTicks = 10 * TPS
	DawnTicks  = 5 * TPS
)

// What the world's colors are multiplied by at full night, from white by day
var NightTint = color.RGBA{110, 125, 180, 255}
