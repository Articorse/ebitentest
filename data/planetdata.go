package data

// Saved worlds refer to their planet by these values, so they must never change
type PlanetEnum uint16

const (
	Planet_None PlanetEnum = iota
	Planet_Earthlike
	Planet_Moon
)

const (
	// What a world saved before it had a planet is on
	DefaultPlanet = Planet_Earthlike
	SeasonDays    = 7
	YearDays      = 4 * SeasonDays
	// A year starts at the start of spring, so the middle of summer is three eighths of the way through it
	MidsummerYearFraction = 3.0 / 8
	// The size of the bumps the noise puts in the north-south curve
	ClimateNoiseScaleTiles = 1000.0
)
