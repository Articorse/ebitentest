package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"fmt"
	"slices"
)

type EquipableSlotEnum uint8

const (
	Equipable_MainHand = EquipableSlotEnum(1 << Equip_MainHand)
	Equipable_OffHand  = EquipableSlotEnum(1 << Equip_OffHand)
	Equipable_Torso    = EquipableSlotEnum(1 << Equip_Torso)
	Equipable_Arms     = EquipableSlotEnum(1 << Equip_Arms)
	Equipable_Legs     = EquipableSlotEnum(1 << Equip_Legs)
	Equipable_Ring1    = EquipableSlotEnum(1 << Equip_Ring1)
	Equipable_Ring2    = EquipableSlotEnum(1 << Equip_Ring2)
)

type Equipment struct {
	slot      EquipableSlotEnum
	abilities [data.MaxEquipmentAbilitySlots]EntityAbility
	// The kinds of body that can equip it, or any when empty
	bodies []data.BodyEnum
	// Drawn over whoever wears it, at the frame its sprite shows, or nothing if none
	worn common.ImageAssetTag
}

func NewEquipmentComponent(slot EquipableSlotEnum, abilities [data.MaxEquipmentAbilitySlots]EntityAbility) *Equipment {
	abis := [data.MaxEquipmentAbilitySlots]EntityAbility{}

	for i, abi := range abilities {
		abis[i] = EntityAbility{
			Def:    abi.Def,
			Status: AbilityStatus{State: AbiAct_Ready},
			Params: abi.Params,
		}
	}

	return &Equipment{
		slot:      slot,
		abilities: abis,
	}
}

// Equipment with no abilities, such as clothing, which only bodies of the given kinds can wear
func NewWornEquipmentComponent(slot EquipableSlotEnum, bodies []data.BodyEnum, worn common.ImageAssetTag) *Equipment {
	return &Equipment{slot: slot, bodies: slices.Clone(bodies), worn: worn}
}

func (x *Equipment) FitsBody(kind data.BodyEnum) bool {
	return len(x.bodies) == 0 || slices.Contains(x.bodies, kind)
}

func (x *Equipment) Worn() common.ImageAssetTag {
	return x.worn
}

func (x *Equipment) HasAbility(id AbilityEnum) bool {
	return slices.ContainsFunc(x.abilities[:], func(a EntityAbility) bool { return a.Def.AbilityId == id })
}

func (x *Equipment) Fits(slot EquipSlotEnum) bool {
	return x.slot&(1<<slot) != 0
}

func (x *ECSContainer) ActivateEquipmentAbility(
	e common.EntityId,
	slot EquipSlotEnum,
	target AbilityTarget,
	abiIdx int,
) (activated bool, err error) {
	if abiIdx > data.MaxEquipmentAbilitySlots-1 {
		return false, fmt.Errorf("ability index %d is out of bounds", abiIdx)
	}

	equipper, ok := x.Equippers.Get(e)
	if !ok {
		return false, fmt.Errorf("entity %d has no equipper", e)
	}

	eqE, ok := equipper.InSlot(slot)
	if !ok || x.IsDisabled(e) || x.IsDisabled(eqE) {
		return false, nil
	}

	eq, ok := x.Equipments.Get(eqE)
	if !ok {
		return false, fmt.Errorf("entity %d in slot %v of entity %d has no equipment", eqE, slot, e)
	}

	// Activated on a copy, so that a failed activation leaves the ability as it was
	abi := eq.abilities[abiIdx]
	activated, err = tryActivate(eqE, &abi, target, x)
	if err != nil {
		return false, fmt.Errorf("error trying to activate ability %v of equipment entity %d: %v", abi.Def.AbilityId, eqE, err)
	}
	eq.abilities[abiIdx] = abi

	return activated, nil
}

func (x *ECSContainer) TickEquipmentAbilities(e common.EntityId) error {
	eq, ok := x.Equipments.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no equipment", e)
	}

	for i, a := range eq.abilities {
		if err := tickAbilityState(e, &a, x); err != nil {
			return fmt.Errorf("error ticking ability %v of equipment entity %d: %v", a.Def.AbilityId, e, err)
		}
		eq.abilities[i] = a
	}

	return nil
}

func (x Equipment) Copy() Equipment {
	copyAbilityParams(x.abilities[:])
	x.bodies = slices.Clone(x.bodies)
	return x
}

func (x Equipment) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Equipments.addComponent(e, x.Copy())
}

type equipmentDto struct {
	Slot      EquipableSlotEnum
	Abilities [data.MaxEquipmentAbilitySlots]EntityAbility
	Bodies    []data.BodyEnum
	Worn      common.ImageAssetTag
}

func (x Equipment) toDto() (ComponentDto, error) {
	copyAbilityParams(x.abilities[:])
	return &equipmentDto{
		Slot:      x.slot,
		Abilities: x.abilities,
		Bodies:    slices.Clone(x.bodies),
		Worn:      x.worn,
	}, nil
}

func (x equipmentDto) toComponent() (Component, error) {
	copyAbilityParams(x.Abilities[:])
	return &Equipment{
		slot:      x.Slot,
		abilities: x.Abilities,
		bodies:    slices.Clone(x.Bodies),
		worn:      x.Worn,
	}, nil
}
