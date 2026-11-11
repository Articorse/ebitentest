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
	MilligramsPerKg   = 1000 * MilligramsPerGram
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
	// The temperature overlay is blue from the first down, green at the second and red from the third up
	OverlayColdCelsius = 0.0
	OverlayMildCelsius = 20.0
	OverlayHotCelsius  = 40.0
	// Air with less oxygen or more carbon dioxide than this per tile is bad to breathe
	SafeOxygenMg        = 400 * MilligramsPerGram
	SafeCarbonDioxideMg = 250 * MilligramsPerGram
	// How much of a gas below where it condenses drops out of the air in a step, and how much of a stack of its item
	// above it goes back in, but no less than the least, or all of it if there's less, so that the last of it doesn't
	// take a sixteenth of nearly nothing at a time
	CondenseFractionDen = 16
	BoilFractionDen     = 16
	CondenseLeastMg     = 10 * MilligramsPerGram
	BoilLeastMg         = 10 * MilligramsPerGram
	// What can still be breathed, for where to get to when there's no safe air to get to
	BreathableOxygenMg        = 100 * MilligramsPerGram
	BreathableCarbonDioxideMg = 500 * MilligramsPerGram
	// A spacesuit holds as much air as 10 tiles, so that its 10 kg of oxygen is a tile's worth of air at about Earthlike
	// pressure
	SpacesuitAirTiles = 10
	SpacesuitOxygenMg = 10 * MilligramsPerKg
	SpacesuitCelsius  = 20
	// Per side, in mW per degree: worn in 15 °C Earthlike air, its wearer's heat keeps it at about 30 °C
	SpacesuitConductance = 10_000
	// What heat passes through between air sealed in something and the air around it
	SealedAirSides = 4
	// How many seconds pass for heat in a second of play, as a 24 minute day is 24 hours, so that it moves through
	// walls and air at about the rate it would
	HeatTimeScale = 60
)
