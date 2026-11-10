package templates

import (
	"ebittest/content/inputsources"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/shapes"
	"ebittest/utils"
	"fmt"
)

// A colonist at the colony center, tethered to it, with the player's body, a stack of wood and an axe. It only picks up
// what it's told to, and only uses its hotbar for what it's using at the moment, so it starts with an empty hand.
func SpawnColonist(ecsContainer *ecs.ECSContainer, colony common.EntityId) (common.EntityId, error) {
	pos, err := ecsContainer.WorldPos(colony)
	if err != nil {
		return -1, fmt.Errorf("error getting the position of colony %d for a colonist: %w", colony, err)
	}
	return spawnColonist(ecsContainer, colony, pos, common.AssetSheetGoldSlime, nil, nil)
}

// A colonist with the giant slime's sheet and colliders, at the position, for trying out bodies bigger than a tile
func SpawnGiantColonist(ecsContainer *ecs.ECSContainer, colony common.EntityId, pos utils.Vec2f) (common.EntityId, error) {
	physicsShape, err := GiantSlimePhysicsShape()
	if err != nil {
		return -1, err
	}
	hitboxShape, err := shapes.NewPolygonShape(giantSlimeOutline, utils.Vec2f{})
	if err != nil {
		return -1, err
	}
	return spawnColonist(ecsContainer, colony, pos, common.AssetSheetGiantSlime, physicsShape, hitboxShape)
}

func spawnColonist(
	ecsContainer *ecs.ECSContainer,
	colony common.EntityId,
	pos utils.Vec2f,
	sheet common.ImageAssetTag,
	physicsShape, hitboxShape shapes.Shape,
) (common.EntityId, error) {
	spr, err := ecs.NewSpriteComponent(sheet, 20, false)
	if err != nil {
		return -1, err
	}
	anim, physics, hitbox, err := colonySlimeParts(sheet, physicsShape, hitboxShape)
	if err != nil {
		return -1, err
	}
	inv := ecs.NewInventoryComponent(data.ColonistInventorySlots, data.ColonistInventorySlots, data.ColonistMaxCarryMg)
	inv.SetPickupsOffHotbar(true)

	colonist, err := ecsContainer.AddNewEntity(
		ecs.NewHierarchyComponent(),
		ecs.NewTransformComponent(pos, 1, 0),
		ecs.NewVelocityComponentWithParams(utils.Vec2f{}, data.ColonistAcceleration, data.DefaultDrag),
		spr,
		anim,
		physics,
		hitbox,
		ecs.NewBodyComponent(data.Body_Humanoid),
		ecs.NewFactionComponent(data.Faction_Colony),
		ecs.NewInvulnerabilityComponent(1000),
		ecs.NewPersistentComponent(),
		inv,
		ecs.NewEquipperComponent(map[ecs.EquipSlotEnum]common.EntityId{}),
		ecs.NewAimComponent(),
		ecs.NewConstructorComponent(data.PlayerBuildRangeTiles, data.PlayerBuildWork, data.PlayerBuildCooldownMs),
		ecs.NewTetherComponent(colony),
		ecs.NewInputComponent(nil, inputsources.InputType_Colonist, &inputsources.InputColonistParams{Colony: colony}, ecs.Facing_None),
	)
	if err != nil {
		return -1, fmt.Errorf("error spawning a colonist: %w", err)
	}

	wood, err := ecsContainer.SpawnItemStack(data.Item_Wood, data.ColonistStartingWood, pos)
	if err != nil {
		return -1, fmt.Errorf("error spawning the wood of colonist %d: %w", colonist, err)
	}
	if err := ecsContainer.PutInInventory(colonist, 0, wood); err != nil {
		return -1, fmt.Errorf("error giving colonist %d its wood: %w", colonist, err)
	}
	axe, err := spawnAxe(ecsContainer, pos)
	if err != nil {
		return -1, fmt.Errorf("error spawning the axe of colonist %d: %w", colonist, err)
	}
	// In the last slot, out of the way of what it picks up
	if err := ecsContainer.PutInInventory(colonist, data.ColonistInventorySlots-1, axe); err != nil {
		return -1, fmt.Errorf("error giving colonist %d its axe: %w", colonist, err)
	}
	return colonist, nil
}
