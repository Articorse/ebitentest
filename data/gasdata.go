package data

// Saved air refers to its gases by these values, so they must never change
type GasEnum uint16

const (
	Gas_None GasEnum = iota
	Gas_Oxygen
	Gas_CarbonDioxide
	GasCount
)

const (
	MilligramsPerGram = 1000
	// How often air flows between rooms and the outdoors
	AirStepTicks = TPS / 4
	// How much of the difference in a gas per tile either side of an opening crosses it in a step, as that many tiles'
	// worth: 5/8 of a tile 4 times a second takes a 25 tile room most of the way to the outdoors' air through one
	// opening in about 30 seconds
	OpeningFlowNum = 5
	OpeningFlowDen = 8
	// The oxygen overlay is red up to the first, yellow halfway and green from the second
	OverlayOxygenRedMg   = 100 * MilligramsPerGram
	OverlayOxygenGreenMg = 500 * MilligramsPerGram
	OverlayOxygenAlpha   = 110
)
