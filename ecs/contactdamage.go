package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"maps"
	"slices"
)

// Hits each entity and tile it touches once, for as long as it exists, unless told otherwise
type ContactDamage struct {
	source          common.EntityId
	knockback       float64
	damageTiers     []int
	dieOnContact    bool
	activeMs        int
	rehitIntervalMs int

	ageMs int
	// How old the source was when it last hit each target. A tile and the feature on it are hit separately.
	hitEntities map[common.EntityId]int
	hitTiles    map[utils.Vec2i]int
	hitFeatures map[utils.Vec2i]int
}

func NewContactDamageComponent(source common.EntityId, knockback float64, damageTiers ...int) *ContactDamage {
	return &ContactDamage{
		source:      source,
		knockback:   knockback,
		damageTiers: damageTiers,
		hitEntities: make(map[common.EntityId]int),
		hitTiles:    make(map[utils.Vec2i]int),
		hitFeatures: make(map[utils.Vec2i]int),
	}
}

// Destroyed by the first entity it hurts or impassable tile it touches
func (x *ContactDamage) WithDieOnContact() *ContactDamage {
	x.dieOnContact = true
	return x
}

// Only hurts during the given time after it's created
func (x *ContactDamage) WithActiveMs(ms int) *ContactDamage {
	x.activeMs = ms
	return x
}

// Hurts what it keeps touching again every interval
func (x *ContactDamage) WithRehitIntervalMs(ms int) *ContactDamage {
	x.rehitIntervalMs = ms
	return x
}

func (x *ContactDamage) Source() common.EntityId {
	return x.source
}

// A copy, so that callers can't change the tiers of the component
func (x *ContactDamage) DamageTiers() []int {
	return slices.Clone(x.damageTiers)
}

func (x *ContactDamage) Knockback() float64 {
	return x.knockback
}

func (x *ContactDamage) DieOnContact() bool {
	return x.dieOnContact
}

func (x *ContactDamage) IsActive() bool {
	return x.activeMs == 0 || x.ageMs < x.activeMs
}

func (x *ContactDamage) CanHitEntity(e common.EntityId) bool {
	lastHit, hit := x.hitEntities[e]
	return x.canHit(lastHit, hit)
}

func (x *ContactDamage) CanHitTile(t utils.Vec2i) bool {
	lastHit, hit := x.hitTiles[t]
	return x.canHit(lastHit, hit)
}

func (x *ContactDamage) CanHitFeature(t utils.Vec2i) bool {
	lastHit, hit := x.hitFeatures[t]
	return x.canHit(lastHit, hit)
}

func (x *ContactDamage) canHit(lastHit int, hit bool) bool {
	return x.IsActive() && (!hit || (x.rehitIntervalMs > 0 && x.ageMs-lastHit >= x.rehitIntervalMs))
}

func (x *ContactDamage) RecordEntityHit(e common.EntityId) {
	x.hitEntities[e] = x.ageMs
}

func (x *ContactDamage) RecordTileHit(t utils.Vec2i) {
	x.hitTiles[t] = x.ageMs
}

func (x *ContactDamage) RecordFeatureHit(t utils.Vec2i) {
	x.hitFeatures[t] = x.ageMs
}

func (x *ContactDamage) TickAge() {
	x.ageMs += data.TickMs
}

func (x ContactDamage) Copy() ContactDamage {
	x.damageTiers = slices.Clone(x.damageTiers)
	x.hitEntities = maps.Clone(x.hitEntities)
	x.hitTiles = maps.Clone(x.hitTiles)
	x.hitFeatures = maps.Clone(x.hitFeatures)
	if x.hitEntities == nil {
		x.hitEntities = make(map[common.EntityId]int)
	}
	if x.hitTiles == nil {
		x.hitTiles = make(map[utils.Vec2i]int)
	}
	if x.hitFeatures == nil {
		x.hitFeatures = make(map[utils.Vec2i]int)
	}
	return x
}

func (x ContactDamage) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.ContactDamages.addComponent(e, x.Copy())
}

type contactDamageDto struct {
	Source          common.EntityId
	Knockback       float64
	DamageTiers     []int
	DieOnContact    bool
	ActiveMs        int
	RehitIntervalMs int
	AgeMs           int
	HitEntities     map[common.EntityId]int
	HitTiles        map[utils.Vec2i]int
	HitFeatures     map[utils.Vec2i]int
}

func (x ContactDamage) toDto() (ComponentDto, error) {
	x = x.Copy()
	return &contactDamageDto{
		Source:          x.source,
		Knockback:       x.knockback,
		DamageTiers:     x.damageTiers,
		DieOnContact:    x.dieOnContact,
		ActiveMs:        x.activeMs,
		RehitIntervalMs: x.rehitIntervalMs,
		AgeMs:           x.ageMs,
		HitEntities:     x.hitEntities,
		HitTiles:        x.hitTiles,
		HitFeatures:     x.hitFeatures,
	}, nil
}

func (x contactDamageDto) toComponent() (Component, error) {
	c := ContactDamage{
		source:          x.Source,
		knockback:       x.Knockback,
		damageTiers:     x.DamageTiers,
		dieOnContact:    x.DieOnContact,
		activeMs:        x.ActiveMs,
		rehitIntervalMs: x.RehitIntervalMs,
		ageMs:           x.AgeMs,
		hitEntities:     x.HitEntities,
		hitTiles:        x.HitTiles,
		hitFeatures:     x.HitFeatures,
	}.Copy()
	return &c, nil
}
