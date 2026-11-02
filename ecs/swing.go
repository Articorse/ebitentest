package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

// Swings a held item clockwise when its first ability is used, then brings it back over the rest of the cooldown.
// The pose is worked out from the ability's cooldown, so it only needs to know which side the holder aims at.
type Swing struct {
	angle float64
	// How much of the cooldown the swing itself takes; the return takes the rest
	swingShare float64
	// Pixel in the sprite's image that stays in place, like the end of a handle
	grip     utils.Vec2f
	mirrored bool
}

func NewSwingComponent(angle float64, swingShare float64, grip utils.Vec2f) *Swing {
	return &Swing{angle: angle, swingShare: swingShare, grip: grip}
}

func (x *Swing) Grip() utils.Vec2f {
	return x.grip
}

// Mirrored items are flipped and swing counter-clockwise
func (x *Swing) Mirrored() bool {
	return x.mirrored
}

func (x *Swing) SetMirrored(mirrored bool) {
	x.mirrored = mirrored
}

// From the moment the item is used until it's ready again
func (x *ECSContainer) IsSwinging(e common.EntityId) bool {
	eq, ok := x.Equipments.Get(e)
	if !ok || !x.Swings.HasComponent(e) {
		return false
	}

	state := eq.abilities[0].Status.State
	return state == AbiAct_Active || state == AbiAct_OnCooldown
}

// Clockwise, in radians, and not mirrored
func (x *ECSContainer) SwingAngle(e common.EntityId) float64 {
	sw, swOk := x.Swings.Get(e)
	eq, eqOk := x.Equipments.Get(e)
	if !swOk || !eqOk {
		return 0
	}

	status := eq.abilities[0].Status
	cooldown := eq.abilities[0].Def.CooldownMs
	if status.State != AbiAct_OnCooldown || cooldown <= 0 {
		return 0
	}

	progress := float64(cooldown-status.CooldownCounterMs) / float64(cooldown)
	if progress < sw.swingShare {
		return sw.angle * progress / sw.swingShare
	}
	return sw.angle * (1 - progress) / (1 - sw.swingShare)
}

func (x Swing) Copy() Swing {
	return x
}

func (x Swing) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Swings.addComponent(e, x.Copy())
}

type swingDto struct {
	Angle      float64
	SwingShare float64
	Grip       utils.Vec2f
	Mirrored   bool
}

func (x Swing) toDto() (ComponentDto, error) {
	return &swingDto{Angle: x.angle, SwingShare: x.swingShare, Grip: x.grip, Mirrored: x.mirrored}, nil
}

func (x swingDto) toComponent() (Component, error) {
	return &Swing{angle: x.Angle, swingShare: x.SwingShare, grip: x.Grip, mirrored: x.Mirrored}, nil
}
