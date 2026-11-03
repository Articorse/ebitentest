package ecs

import (
	"ebittest/data"
	"encoding/json"
	"fmt"
	"os"
)

// What every body of a kind is made of, and the capacities its parts govern
type BodyAtlas struct {
	capacities map[data.CapacityEnum]capacityDef
	bodies     map[data.BodyEnum]bodyDef
}

type capacityDef struct {
	name string
	// The least the capacity drops to however hurt its parts are, such as crawling without legs
	floor int
}

type bodyDef struct {
	name string
	// In the atlas's order, which is the order hits are rolled in
	parts []partDef
	// Killing the body when any of them drops to 0
	vitalCapacities []data.CapacityEnum
}

type partDef struct {
	name      string
	maxHealth int
	// How likely a hit is to land on it, relative to the body's other intact parts
	coverage int
	// Destroying it kills the body
	vital      bool
	capacities []data.CapacityEnum
}

type bodyAtlasDTO struct {
	Capacities []capacityDefDTO `json:"capacities"`
	Bodies     []bodyDefDTO     `json:"bodies"`
}

type capacityDefDTO struct {
	Id           uint16 `json:"id"`
	Name         string `json:"name"`
	FloorPercent int    `json:"floorPercent"`
}

type bodyDefDTO struct {
	Id              uint16       `json:"id"`
	Name            string       `json:"name"`
	Parts           []partDefDTO `json:"parts"`
	VitalCapacities []uint16     `json:"vitalCapacities"`
}

type partDefDTO struct {
	Name       string   `json:"name"`
	MaxHealth  int      `json:"maxHealth"`
	Coverage   int      `json:"coverage"`
	Vital      bool     `json:"vital"`
	Capacities []uint16 `json:"capacities"`
}

func LoadBodyAtlasFromJson(path string) (*BodyAtlas, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading body atlas json file: %w", err)
	}

	var dto bodyAtlasDTO
	if err := json.Unmarshal(f, &dto); err != nil {
		return nil, fmt.Errorf("error unmarshalling body atlas json: %w", err)
	}

	atlas := &BodyAtlas{capacities: map[data.CapacityEnum]capacityDef{}, bodies: map[data.BodyEnum]bodyDef{}}
	for _, c := range dto.Capacities {
		id := data.CapacityEnum(c.Id)
		switch _, taken := atlas.capacities[id]; {
		case id == data.Capacity_None:
			return nil, fmt.Errorf("capacity %q has id 0, which means none", c.Name)
		case taken:
			return nil, fmt.Errorf("capacity id %d is defined twice", c.Id)
		case c.FloorPercent < 0 || c.FloorPercent > 100:
			return nil, fmt.Errorf("capacity %d has a floor of %d%%, want 0 to 100", c.Id, c.FloorPercent)
		}
		atlas.capacities[id] = capacityDef{name: c.Name, floor: c.FloorPercent * data.FullEfficiency / 100}
	}

	capacities := func(ids []uint16) ([]data.CapacityEnum, error) {
		out := make([]data.CapacityEnum, len(ids))
		for i, id := range ids {
			if _, ok := atlas.capacities[data.CapacityEnum(id)]; !ok {
				return nil, fmt.Errorf("capacity %d isn't defined", id)
			}
			out[i] = data.CapacityEnum(id)
		}
		return out, nil
	}

	for _, b := range dto.Bodies {
		id := data.BodyEnum(b.Id)
		switch _, taken := atlas.bodies[id]; {
		case id == data.Body_None:
			return nil, fmt.Errorf("body %q has id 0, which means none", b.Name)
		case taken:
			return nil, fmt.Errorf("body id %d is defined twice", b.Id)
		}

		def := bodyDef{name: b.Name}
		names := map[string]bool{}
		totalCoverage := 0
		for _, p := range b.Parts {
			switch {
			case p.Name == "" || names[p.Name]:
				return nil, fmt.Errorf("body %d has a part without a name of its own, %q", b.Id, p.Name)
			case p.MaxHealth <= 0:
				return nil, fmt.Errorf("part %q of body %d has %d max health, want it positive", p.Name, b.Id, p.MaxHealth)
			case p.Coverage < 0:
				return nil, fmt.Errorf("part %q of body %d has a negative coverage", p.Name, b.Id)
			}
			names[p.Name] = true
			totalCoverage += p.Coverage

			caps, err := capacities(p.Capacities)
			if err != nil {
				return nil, fmt.Errorf("part %q of body %d: %w", p.Name, b.Id, err)
			}
			def.parts = append(def.parts, partDef{name: p.Name, maxHealth: p.MaxHealth, coverage: p.Coverage, vital: p.Vital, capacities: caps})
		}
		if totalCoverage == 0 {
			return nil, fmt.Errorf("body %d has no part that can be hit", b.Id)
		}

		vital, err := capacities(b.VitalCapacities)
		if err != nil {
			return nil, fmt.Errorf("body %d's vital capacities: %w", b.Id, err)
		}
		def.vitalCapacities = vital
		atlas.bodies[id] = def
	}

	return atlas, nil
}

func (x *ECSContainer) SetBodyAtlas(atlas *BodyAtlas) {
	x.bodyAtlas = atlas
}

// Each capacity is the average efficiency of the parts that govern it, but never below its floor. A capacity no part
// governs is at full efficiency, so a body without legs isn't slowed by not having any.
func (def bodyDef) capacities(atlas *BodyAtlas, health map[string]int) map[data.CapacityEnum]int {
	sums, counts := map[data.CapacityEnum]int{}, map[data.CapacityEnum]int{}
	for _, p := range def.parts {
		efficiency := def.partHealth(p, health) * data.FullEfficiency / p.maxHealth
		for _, c := range p.capacities {
			sums[c] += efficiency
			counts[c]++
		}
	}

	out := make(map[data.CapacityEnum]int, len(sums))
	for c, sum := range sums {
		out[c] = max(sum/counts[c], atlas.capacities[c].floor)
	}
	return out
}

// Only damaged parts are in health
func (def bodyDef) partHealth(p partDef, health map[string]int) int {
	if h, ok := health[p.name]; ok {
		return h
	}
	return p.maxHealth
}
