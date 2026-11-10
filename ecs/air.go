package ecs

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"math"
)

// The tile whose air something at pos breathes: the one it's on, or on a door, which has none, the one beside it on
// whichever side it's furthest over towards
func (x *ECSContainer) AirTile(pos utils.Vec2f) utils.Vec2i {
	t := utils.WorldPosToTilePos(pos)
	if _, ok := x.terrain.AirAt(t); ok {
		return t
	}
	off := pos.Subtract(utils.TilePosToWorldPos(t))
	sign := func(v float64) int {
		if v < 0 {
			return -1
		}
		return 1
	}
	sides := []utils.Vec2i{{X: t.X + sign(off.X), Y: t.Y}, {X: t.X, Y: t.Y + sign(off.Y)}}
	if math.Abs(off.Y) > math.Abs(off.X) {
		sides[0], sides[1] = sides[1], sides[0]
	}
	for _, s := range sides {
		if _, ok := x.terrain.AirAt(s); ok {
			return s
		}
	}
	return t
}

// The air per tile the entity breathes, which is what it wears that has air sealed in it if it does, false where there's
// none
func (x *ECSContainer) AirOf(e common.EntityId) (climate.Air, bool) {
	if sealed, ok := x.WornSealedAir(e); ok {
		return sealed.PerTile(), true
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return climate.Air{}, false
	}
	return x.terrain.AirAt(x.AirTile(pos))
}

// Enough oxygen and little enough carbon dioxide not to be bad to breathe
func SafeAir(a climate.Air) bool {
	return a[data.Gas_Oxygen] >= data.SafeOxygenMg && a[data.Gas_CarbonDioxide] <= data.SafeCarbonDioxideMg
}

// Air that's bad to breathe, but can still be breathed
func BreathableAir(a climate.Air) bool {
	return a[data.Gas_Oxygen] >= data.BreathableOxygenMg && a[data.Gas_CarbonDioxide] <= data.BreathableCarbonDioxideMg
}

// Whether the entity breathes and what it's breathing is bad for it, which is shown over it
func (x *ECSContainer) InBadAir(e common.EntityId) bool {
	if x.OxygenPerSecond(e) == 0 {
		return false
	}
	air, ok := x.AirOf(e)
	return ok && !SafeAir(air)
}
