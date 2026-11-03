package data

// The day starts at tick 0 and runs day, dusk, night, dawn
const (
	DayTicks   = 10 * TPS
	DuskTicks  = 5 * TPS
	NightTicks = 10 * TPS
	DawnTicks  = 5 * TPS
	// At night, visions reach this fraction of their radius, rounded down
	NightVisionDivisor = 3
)
