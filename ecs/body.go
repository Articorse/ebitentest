package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"maps"
	"slices"
)

// Parts that take damage, as the body atlas defines them for the body's kind, and the capacities they govern
type Body struct {
	kind data.BodyEnum
	// Only for damaged parts, by name
	health map[string]int
	// Worked out from the parts and conditions whenever they change, so it isn't saved
	capacities map[data.CapacityEnum]int
	conditions map[data.ConditionEnum]*condition
}

func NewBodyComponent(kind data.BodyEnum) *Body {
	return &Body{kind: kind, health: map[string]int{}}
}

func (x *Body) Kind() data.BodyEnum {
	return x.kind
}

// In thousandths, data.FullEfficiency for entities without a body, which nothing limits
func (x *ECSContainer) Capacity(e common.EntityId, capacity data.CapacityEnum) int {
	body, ok := x.Bodies.Get(e)
	if !ok {
		return data.FullEfficiency
	}
	if c, ok := body.capacities[capacity]; ok {
		return c
	}
	return data.FullEfficiency
}

// How much oxygen the entity breathes in a second, and as much carbon dioxide out, 0 without a body
func (x *ECSContainer) OxygenPerSecond(e common.EntityId) int64 {
	_, def, err := x.body(e)
	if err != nil {
		return 0
	}
	return def.oxygenMgPerSecond
}

// How much heat the entity gives off into the air it's in, 0 without a body
func (x *ECSContainer) HeatWatts(e common.EntityId) int64 {
	_, def, err := x.body(e)
	if err != nil {
		return 0
	}
	return def.heatWatts
}

func (x *ECSContainer) PartHealth(e common.EntityId, part string) (health, maxHealth int, ok bool) {
	body, def, err := x.body(e)
	if err != nil {
		return 0, 0, false
	}
	for _, p := range def.parts {
		if p.name == part {
			return def.partHealth(p, body.health), p.maxHealth, true
		}
	}
	return 0, 0, false
}

func (x *ECSContainer) body(e common.EntityId) (*Body, bodyDef, error) {
	body, ok := x.Bodies.Get(e)
	if !ok {
		return nil, bodyDef{}, fmt.Errorf("entity %d has no body", e)
	}
	if x.bodyAtlas == nil {
		return nil, bodyDef{}, fmt.Errorf("entity %d has a body, but there's no body atlas", e)
	}
	def, ok := x.bodyAtlas.bodies[body.kind]
	if !ok {
		return nil, bodyDef{}, fmt.Errorf("entity %d's body is of kind %v, which isn't in the body atlas", e, body.kind)
	}
	return body, def, nil
}

// Hurts one of e's intact parts, rolled by their coverage, and makes e invulnerable and flash for as long as its
// invulnerability says. Damage beyond what the part has left is lost. Dead once a vital part or a vital capacity drops
// to nothing.
func (x *ECSContainer) Damage(e common.EntityId, damage int) (dead bool, err error) {
	body, def, err := x.body(e)
	if err != nil {
		return false, err
	}

	total := 0
	for _, p := range def.parts {
		if def.partHealth(p, body.health) > 0 {
			total += p.coverage
		}
	}
	if total == 0 {
		return false, nil
	}

	roll := x.Rng.IntN(total)
	var hit partDef
	for _, p := range def.parts {
		if def.partHealth(p, body.health) <= 0 {
			continue
		}
		if roll < p.coverage {
			hit = p
			break
		}
		roll -= p.coverage
	}

	health := max(def.partHealth(hit, body.health)-damage, 0)
	body.health[hit.name] = health
	body.capacities = x.capacitiesOf(body, def)

	postHitMs := 0
	if inv, ok := x.Invulnerabilities.Get(e); ok {
		postHitMs = inv.postHitMs
		inv.remainingMs = max(inv.remainingMs, postHitMs)
	}

	if hit.vital && health == 0 {
		return true, nil
	}
	for _, c := range def.vitalCapacities {
		if x.Capacity(e, c) == 0 {
			return true, nil
		}
	}

	if s, ok := x.Sprites.Get(e); ok {
		err := s.SetFlash(
			[]utils.RelativeColor{
				{R: 0.3, G: 0.3, B: 0.3, A: 1},
				{R: 1, G: 1, B: 1, A: 1},
			},
			[]int{100, 100},
			postHitMs,
		)
		if err != nil {
			return false, fmt.Errorf("could not set sprite flash of entity %d: %v", e, err)
		}
	}

	return false, nil
}

func (x Body) Copy() Body {
	conditions := map[data.ConditionEnum]*condition{}
	for id, c := range x.conditions {
		copied := *c
		conditions[id] = &copied
	}
	return Body{kind: x.kind, health: maps.Clone(x.health), capacities: maps.Clone(x.capacities), conditions: conditions}
}

// The capacities are worked out from the atlas whenever the body is added or loaded
func (x Body) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	c := x.Copy()
	if c.health == nil {
		c.health = map[string]int{}
	}
	ecsContainer.Bodies.addComponent(e, c)

	body, def, err := ecsContainer.body(e)
	if err != nil {
		log.Printf("error working out the capacities of entity %d: %v", e, err)
		return
	}
	body.capacities = ecsContainer.capacitiesOf(body, def)
}

type bodyDto struct {
	Kind       data.BodyEnum
	Health     map[string]int
	Conditions map[data.ConditionEnum]conditionDto
}

type conditionDto struct {
	Severity, Damage int64
	Worsening        bool
}

func (x Body) toDto() (ComponentDto, error) {
	conditions := map[data.ConditionEnum]conditionDto{}
	for id, c := range x.conditions {
		conditions[id] = conditionDto{Severity: c.severity, Damage: c.damage, Worsening: c.worsening}
	}
	return &bodyDto{Kind: x.kind, Health: maps.Clone(x.health), Conditions: conditions}, nil
}

func (x bodyDto) toComponent() (Component, error) {
	conditions := map[data.ConditionEnum]*condition{}
	for id, c := range x.Conditions {
		conditions[id] = &condition{severity: c.Severity, damage: c.Damage, worsening: c.Worsening}
	}
	return &Body{kind: x.Kind, health: maps.Clone(x.Health), conditions: conditions}, nil
}

type PartStatus struct {
	Name              string
	Health, MaxHealth int
}

type CapacityStatus struct {
	Name string
	// In thousandths
	Efficiency int
}

// The parts in the body atlas's order, and the capacities they govern by id
func (x *ECSContainer) BodyStatus(e common.EntityId) ([]PartStatus, []CapacityStatus, error) {
	body, def, err := x.body(e)
	if err != nil {
		return nil, nil, err
	}

	parts := make([]PartStatus, len(def.parts))
	for i, p := range def.parts {
		parts[i] = PartStatus{Name: p.name, Health: def.partHealth(p, body.health), MaxHealth: p.maxHealth}
	}

	ids := slices.Sorted(maps.Keys(body.capacities))
	capacities := make([]CapacityStatus, len(ids))
	for i, id := range ids {
		capacities[i] = CapacityStatus{Name: x.bodyAtlas.capacities[id].name, Efficiency: body.capacities[id]}
	}
	return parts, capacities, nil
}
