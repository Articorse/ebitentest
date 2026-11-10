package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math/rand/v2"
	"slices"
)

type ItemDrop struct {
	Kind   data.ItemEnum
	Amount int
	// Anywhere from Amount to it, evenly, if it's more than Amount
	Max int
}

type DropOutcome struct {
	ChancePercent float64
	Items         []ItemDrop
}

// At most one outcome is rolled each time. Their chances add up to at most 100%, and the rest is the chance of
// dropping nothing.
type DropTable struct {
	outcomes []DropOutcome
}

func NewDropTable(outcomes ...DropOutcome) (*DropTable, error) {
	total := 0.0
	for i, o := range outcomes {
		if o.ChancePercent < 0 {
			return nil, fmt.Errorf("drop outcome %d has a negative chance, %v%%", i, o.ChancePercent)
		}
		for _, item := range o.Items {
			if item.Amount < 0 || max(item.Amount, item.Max) < 1 || (item.Max != 0 && item.Max < item.Amount) {
				return nil, fmt.Errorf("drop outcome %d drops %d to %d of item kind %v, want at least some", i, item.Amount, item.Max, item.Kind)
			}
		}
		total += o.ChancePercent
	}
	if total > 100 {
		return nil, fmt.Errorf("drop outcomes add up to %v%%, more than 100%%", total)
	}

	return (&DropTable{outcomes: outcomes}).Copy(), nil
}

// What's dropped, each of its items rolled from its range, leaving out what comes to none
func (x *DropTable) Roll(rng *rand.Rand) []ItemDrop {
	r := rng.Float64() * 100
	for _, o := range x.outcomes {
		if r >= o.ChancePercent {
			r -= o.ChancePercent
			continue
		}
		drops := []ItemDrop{}
		for _, item := range o.Items {
			if item.Max > item.Amount {
				item.Amount += rng.IntN(item.Max - item.Amount + 1)
			}
			if item.Amount > 0 {
				drops = append(drops, ItemDrop{Kind: item.Kind, Amount: item.Amount})
			}
		}
		return drops
	}
	return nil
}

// Rolls the table and lays down what it gives at the tile, or as close to it as there's room
func (x *ECSContainer) LayDownDrops(table *DropTable, t utils.Vec2i) {
	for _, drop := range table.Roll(x.Rng) {
		if laid := x.LayDownKind(drop.Kind, drop.Amount, t); laid < drop.Amount {
			log.Printf("no room around %v for %d of the %d of item %v dropped there", t, drop.Amount-laid, drop.Amount, drop.Kind)
		}
	}
}

// Damages the tile, and drops what it leaves behind if that destroyed it
func (x *ECSContainer) DamageTile(t utils.Vec2i, damage int) bool {
	damaged, drops := x.terrain.DamageTile(t, damage)
	x.dropFrom(t, drops)
	return damaged
}

// Damages the feature on the tile, and drops what it leaves behind if that destroyed it
func (x *ECSContainer) DamageFeature(t utils.Vec2i, damage int) bool {
	damaged, drops := x.terrain.DamageFeature(t, damage)
	x.dropFrom(t, drops)
	return damaged
}

func (x *ECSContainer) dropFrom(t utils.Vec2i, drops *DropTable) {
	if drops == nil {
		return
	}
	x.LayDownDrops(drops, t)
}

// Whether any of its outcomes drops the kind
func (x *DropTable) CanDrop(kind data.ItemEnum) bool {
	return x != nil && slices.ContainsFunc(x.outcomes, func(o DropOutcome) bool {
		return slices.ContainsFunc(o.Items, func(i ItemDrop) bool { return i.Kind == kind })
	})
}

func (x *DropTable) Copy() *DropTable {
	c := &DropTable{outcomes: slices.Clone(x.outcomes)}
	for i, o := range c.outcomes {
		c.outcomes[i].Items = slices.Clone(o.Items)
	}
	return c
}

func (x DropTable) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.DropTables.addComponent(e, *x.Copy())
}

type dropTableDto struct {
	Outcomes []DropOutcome
}

func (x DropTable) toDto() (ComponentDto, error) {
	return &dropTableDto{Outcomes: x.Copy().outcomes}, nil
}

func (x dropTableDto) toComponent() (Component, error) {
	return (&DropTable{outcomes: x.Outcomes}).Copy(), nil
}
