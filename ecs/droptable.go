package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"slices"
)

type ItemDrop struct {
	Kind  ItemKindEnum
	Count int
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
			if item.Count < 1 {
				return nil, fmt.Errorf("drop outcome %d drops %d of item kind %v, want at least 1", i, item.Count, item.Kind)
			}
		}
		total += o.ChancePercent
	}
	if total > 100 {
		return nil, fmt.Errorf("drop outcomes add up to %v%%, more than 100%%", total)
	}

	return (&DropTable{outcomes: outcomes}).Copy(), nil
}

func (x *DropTable) roll(rng *rand.Rand) []ItemDrop {
	r := rng.Float64() * 100
	for _, o := range x.outcomes {
		if r < o.ChancePercent {
			return o.Items
		}
		r -= o.ChancePercent
	}
	return nil
}

// Rolls the table and tosses each item it gives, one by one, in a random direction from pos
func (x *ECSContainer) SpawnDrops(table *DropTable, pos utils.Vec2f) error {
	for _, drop := range table.roll(x.Rng) {
		for range drop.Count {
			item, err := x.SpawnItem(drop.Kind, pos)
			if err != nil {
				return fmt.Errorf("error dropping item kind %v at %v: %w", drop.Kind, pos, err)
			}
			x.tossItem(item, x.Rng.Float64()*2*math.Pi)
		}
	}
	return nil
}

// Damages the feature on the tile, and drops what it leaves behind if that destroyed it
func (x *ECSContainer) DamageFeature(t utils.Vec2i, damage int) bool {
	damaged, drops := x.terrain.DamageFeature(t, damage)
	if drops != nil {
		if err := x.SpawnDrops(drops, utils.TilePosToWorldPos(t)); err != nil {
			log.Printf("error dropping what the feature at %v left behind: %v", t, err)
		}
	}
	return damaged
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
