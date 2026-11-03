package data

type RoofEnum uint8

const (
	Roof_None RoofEnum = iota
	// What worldgen puts over mountains
	Roof_Rock
)

// How bright what's under a roof is drawn
const RoofShade = 0.75
