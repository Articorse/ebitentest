package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
)

// What a body's surroundings do to one of its parts while they're bad for it
type conditionDef struct {
	name string
	part string
	// What it goes by, in the units the atlas says it is in, made whole: milligrams per tile, or hundredths of a degree
	cause conditionCause
	// It gets worse while the cause is below the first point, or above it if not below
	below bool
	// Only bodies that breathe get it
	breathing bool
	// From the first point, where it barely gets worse, how much worse it gets in a step at each, in millionths of full
	// severity, rising evenly between them and staying at the last past it
	points []conditionPoint
	// How much better it gets in a step once the cause is gone
	recoveryPerStep int64
}

type conditionCause uint8

const (
	cause_Oxygen conditionCause = iota
	cause_CarbonDioxide
	cause_Temperature
	cause_Pressure
)

var conditionCauses = map[string]conditionCause{
	"oxygen": cause_Oxygen, "carbonDioxide": cause_CarbonDioxide, "temperature": cause_Temperature, "pressure": cause_Pressure,
}

type conditionPoint struct {
	at, perStep int64
}

type ConditionAtlas struct {
	conditions map[data.ConditionEnum]conditionDef
}

type conditionDefDto struct {
	Id        uint16 `json:"id"`
	Name      string `json:"name"`
	Part      string `json:"part"`
	Cause     string `json:"cause"`
	Below     bool   `json:"below"`
	Breathing bool   `json:"breathing"`
	Points    []struct {
		At      float64 `json:"at"`
		Seconds float64 `json:"seconds"`
	} `json:"points"`
	RecoverySeconds float64 `json:"recoverySeconds"`
}

func stepsIn(seconds float64) float64 {
	return seconds * data.TPS / data.AirStepTicks
}

// Points are in grams per tile for gases and pressure, and degrees for temperature. The first point's seconds are
// ignored, since it's where it only barely gets worse.
func LoadConditionAtlasFromJson(path string) (*ConditionAtlas, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading conditions json file: %w", err)
	}
	var dtos []conditionDefDto
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling conditions json: %w", err)
	}
	atlas := &ConditionAtlas{conditions: map[data.ConditionEnum]conditionDef{}}
	for _, d := range dtos {
		id := data.ConditionEnum(d.Id)
		cause, known := conditionCauses[d.Cause]
		switch _, taken := atlas.conditions[id]; {
		case id == data.Condition_None:
			return nil, fmt.Errorf("condition %q has id 0, which means none", d.Name)
		case taken:
			return nil, fmt.Errorf("condition id %d is defined twice", d.Id)
		case !known:
			return nil, fmt.Errorf("condition %d has cause %q, which isn't one there is", d.Id, d.Cause)
		case len(d.Points) < 2:
			return nil, fmt.Errorf("condition %d has %d points, want at least 2", d.Id, len(d.Points))
		case d.RecoverySeconds <= 0:
			return nil, fmt.Errorf("condition %d recovers in %v seconds, want it positive", d.Id, d.RecoverySeconds)
		}
		unit := float64(data.MilligramsPerGram)
		if cause == cause_Temperature {
			unit = 100
		}
		def := conditionDef{
			name: d.Name, part: d.Part, cause: cause, below: d.Below, breathing: d.Breathing,
			recoveryPerStep: int64(math.Round(data.FullSeverity / stepsIn(d.RecoverySeconds))),
		}
		for i, p := range d.Points {
			point := conditionPoint{at: int64(math.Round(p.At * unit))}
			if i > 0 {
				if p.Seconds <= 0 {
					return nil, fmt.Errorf("condition %d's point %d takes %v seconds to full severity, want it positive", d.Id, i, p.Seconds)
				}
				if worse := point.at - def.points[i-1].at; (d.Below && worse >= 0) || (!d.Below && worse <= 0) {
					return nil, fmt.Errorf("condition %d's point %d isn't further into what's bad than the one before", d.Id, i)
				}
				point.perStep = int64(math.Round(data.FullSeverity / stepsIn(p.Seconds)))
			}
			def.points = append(def.points, point)
		}
		atlas.conditions[id] = def
	}
	return atlas, nil
}

func (x *ECSContainer) SetConditionAtlas(atlas *ConditionAtlas) {
	x.conditionAtlas = atlas
}

// How much worse it gets in a step with its cause at v, 0 where v isn't bad for it
func (def conditionDef) worsening(v int64) int64 {
	into := func(p conditionPoint) int64 {
		if def.below {
			return p.at - v
		}
		return v - p.at
	}
	if into(def.points[0]) <= 0 {
		return 0
	}
	for i := 1; i < len(def.points); i++ {
		a, b := def.points[i-1], def.points[i]
		if into(b) <= 0 {
			span := into(a) - into(b)
			return a.perStep + (b.perStep-a.perStep)*into(a)/span
		}
	}
	return def.points[len(def.points)-1].perStep
}

type condition struct {
	// In millionths of full severity
	severity int64
	// Of damage to come, in thousandths of health times ticks, so that what a step adds doesn't get rounded away
	damage int64
	// Whether it got worse in the last step
	worsening bool
}

// Works out what each body's surroundings do to it for a step of the air, which is when they change. A body that a
// condition kills is killed.
func (x *ECSContainer) TickConditions() {
	if x.conditionAtlas == nil {
		return
	}
	for _, e := range slices.Sorted(maps.Keys(maps.Collect(Query1(&x.Bodies, Without(&x.Disabled))))) {
		if dead := x.tickConditions(e); dead {
			x.ScheduleKillEntity(e)
		}
	}
}

func (x *ECSContainer) tickConditions(e common.EntityId) (dead bool) {
	body, def, err := x.body(e)
	if err != nil {
		return false
	}
	air, ok := x.AirOf(e)
	if !ok {
		return false
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return false
	}
	temperature := x.terrain.AirTemperature(x.AirTile(pos))
	breathes := def.oxygenMgPerSecond > 0
	causes := map[conditionCause]int64{
		cause_Oxygen: air[data.Gas_Oxygen], cause_CarbonDioxide: air[data.Gas_CarbonDioxide],
		cause_Temperature: int64(temperature), cause_Pressure: air.Total(),
	}
	if body.conditions == nil {
		body.conditions = map[data.ConditionEnum]*condition{}
	}
	for _, id := range slices.Sorted(maps.Keys(x.conditionAtlas.conditions)) {
		cdef := x.conditionAtlas.conditions[id]
		part, hasPart := def.part(cdef.part)
		if !hasPart || (cdef.breathing && !breathes) {
			continue
		}
		c, has := body.conditions[id]
		worse := cdef.worsening(causes[cdef.cause])
		if !has && worse == 0 {
			continue
		}
		if !has {
			c = &condition{}
			body.conditions[id] = c
		}
		c.worsening = worse > 0
		if worse > 0 {
			c.severity = min(data.FullSeverity, c.severity+worse)
		} else {
			c.severity -= cdef.recoveryPerStep
		}
		if c.severity <= 0 {
			delete(body.conditions, id)
			continue
		}
		if c.severity >= data.ConditionDamageFromSeverity {
			past := c.severity - data.ConditionDamageFromSeverity
			milliHealth := data.ConditionDamageMilliHealthFrom + (data.ConditionDamageMilliHealthAtFull-data.ConditionDamageMilliHealthFrom)*past/(data.FullSeverity-data.ConditionDamageFromSeverity)
			c.damage += milliHealth * data.AirStepTicks
		}
		if whole := c.damage / (1000 * data.TPS); whole > 0 {
			c.damage -= whole * 1000 * data.TPS
			health := max(def.partHealth(part, body.health)-int(whole), 0)
			body.health[part.name] = health
			dead = dead || (part.vital && health == 0)
		}
	}
	body.capacities = x.capacitiesOf(body, def)
	for _, c := range def.vitalCapacities {
		dead = dead || x.Capacity(e, c) == 0
	}
	return dead
}

// What its parts allow, scaled by its capability, which its conditions take down, but none below its floor
func (x *ECSContainer) capacitiesOf(body *Body, def bodyDef) map[data.CapacityEnum]int {
	capacities := def.capacities(x.bodyAtlas, body.health)
	if len(body.conditions) == 0 {
		return capacities
	}
	capability := int64(data.FullEfficiency)
	for _, id := range slices.Sorted(maps.Keys(body.conditions)) {
		if over := body.conditions[id].severity - data.ConditionCapabilityFromSeverity; over > 0 {
			left := data.FullSeverity - data.ConditionCapabilityFromSeverity - over
			capability = capability * left / (data.FullSeverity - data.ConditionCapabilityFromSeverity)
		}
	}
	for c, efficiency := range capacities {
		capacities[c] = max(int(int64(efficiency)*capability/data.FullEfficiency), x.bodyAtlas.capacities[c].floor)
	}
	capacities[data.Capacity_Capability] = int(capability)
	return capacities
}

// Whether any of the entity's conditions got worse in the last step, which is shown over it
func (x *ECSContainer) Worsening(e common.EntityId) bool {
	body, ok := x.Bodies.Get(e)
	if !ok {
		return false
	}
	for _, c := range body.conditions {
		if c.worsening {
			return true
		}
	}
	return false
}

type ConditionStatus struct {
	Name, Part string
	// In whole percent
	Percent   int
	Worsening bool
}

// In the condition atlas's order
func (x *ECSContainer) Conditions(e common.EntityId) []ConditionStatus {
	body, ok := x.Bodies.Get(e)
	if !ok || x.conditionAtlas == nil {
		return nil
	}
	out := []ConditionStatus{}
	for _, id := range slices.Sorted(maps.Keys(body.conditions)) {
		c, def := body.conditions[id], x.conditionAtlas.conditions[id]
		out = append(out, ConditionStatus{Name: def.name, Part: def.part, Percent: int(c.severity * 100 / data.FullSeverity), Worsening: c.worsening})
	}
	return out
}
