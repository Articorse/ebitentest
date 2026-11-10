package data

// Saved bodies refer to their kind by these values, so they must never change
type BodyEnum uint16

const (
	Body_None BodyEnum = iota
	Body_Humanoid
	Body_Giant
)

// What a body can do, worked out from its parts. Capacities aren't saved, but the body atlas refers to them by these
// values.
type CapacityEnum uint16

const (
	Capacity_None CapacityEnum = iota
	Capacity_Moving
	Capacity_Sight
	// What every other capacity is scaled by, which conditions take down
	Capacity_Capability
)

// Efficiencies are in thousandths, in integers, so that what parts govern is worked out the same on every machine
const FullEfficiency = 1000

// Saved bodies refer to their conditions by these values, so they must never change
type ConditionEnum uint16

const (
	Condition_None ConditionEnum = iota
)

const (
	// Severity is in millionths, so that what's little more than nothing still adds up
	FullSeverity = 1_000_000
	// Conditions do their part no damage below this, then from 0.05 health a second to 1.5 at full severity
	ConditionDamageFromSeverity      = FullSeverity / 2
	ConditionDamageMilliHealthFrom   = 50
	ConditionDamageMilliHealthAtFull = 1500
	// Above this, each condition takes capability down, to none at full severity
	ConditionCapabilityFromSeverity = FullSeverity * 6 / 10
)
