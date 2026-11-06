package data

// The areas marked out on tiles, for colonists to know where to work and what to do there
type AreaEnum uint8

const (
	Area_None AreaEnum = iota
	Area_Home
	Area_Harvest
	AreaCount
)
