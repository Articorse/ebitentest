package data

// Saved bodies refer to their kind by these values, so they must never change
type BodyEnum uint16

const (
	Body_None BodyEnum = iota
	Body_Humanoid
)

// What a body can do, worked out from its parts. Capacities aren't saved, but the body atlas refers to them by these
// values.
type CapacityEnum uint16

const (
	Capacity_None CapacityEnum = iota
	Capacity_Moving
)

// Efficiencies are in thousandths, in integers, so that what parts govern is worked out the same on every machine
const FullEfficiency = 1000
