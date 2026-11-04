package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"slices"
)

// Who the entity sides with, which decides who it's hostile to
type Faction struct {
	faction data.FactionEnum
}

func NewFactionComponent(faction data.FactionEnum) *Faction {
	return &Faction{faction: faction}
}

func (x *Faction) Faction() data.FactionEnum {
	return x.faction
}

func AreHostile(a, b data.FactionEnum) bool {
	return slices.Contains(data.HostileFactions, [2]data.FactionEnum{a, b}) || slices.Contains(data.HostileFactions, [2]data.FactionEnum{b, a})
}

// Debugging only, so it isn't saved: untargetable entities are never chosen as targets
func (x *ECSContainer) SetUntargetable(e common.EntityId, untargetable bool) {
	if untargetable {
		x.untargetable[e] = true
	} else {
		delete(x.untargetable, e)
	}
}

func (x *ECSContainer) IsUntargetable(e common.EntityId) bool {
	return x.untargetable[e]
}

// The closest entity with a body, of a faction hostile to e's, whose tile e's vision sees. Entities are looked at in
// order, so ties go to the lowest one, and the choice is the same every run.
func (x *ECSContainer) NearestVisibleHostile(e common.EntityId) (common.EntityId, bool) {
	own, ok := x.Factions.Get(e)
	vision, hasVision := x.Visions.Get(e)
	if !ok || !hasVision {
		return -1, false
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return -1, false
	}

	best, bestDist := common.EntityId(-1), 0.0
	for other, f := range Query1(&x.Factions, With(&x.Bodies), Without(&x.Disabled)) {
		if other == e || !AreHostile(own.faction, f.faction) || x.untargetable[other] {
			continue
		}
		otherPos, err := x.WorldPos(other)
		if err != nil || !vision.CanSee(utils.WorldPosToTilePos(otherPos)) {
			continue
		}
		d := otherPos.Subtract(pos)
		dist := d.X*d.X + d.Y*d.Y
		if best == -1 || dist < bestDist {
			best, bestDist = other, dist
		}
	}
	return best, best != -1
}

func (x Faction) Copy() Faction {
	return x
}

func (x Faction) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Factions.addComponent(e, x.Copy())
}

type factionDto struct {
	Faction data.FactionEnum
}

func (x Faction) toDto() (ComponentDto, error) {
	return &factionDto{Faction: x.faction}, nil
}

func (x factionDto) toComponent() (Component, error) {
	return NewFactionComponent(x.Faction), nil
}
