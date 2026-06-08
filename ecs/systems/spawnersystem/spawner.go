package spawnersystem

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
)

func Spawn(
	spawnerEntity common.EntityId,
	world *ecs.World,
) error {
	sm := ecs.SpawnerManager{}
	tm := ecs.TransformManager{}

	comps, err := sm.GetComponents(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting components to spawn for spawner entity %d: %v", spawnerEntity, err)
	}

	newEntity := world.AddEntity(
		comps...,
	)

	worldPos, _ := tm.GetWorldPos(spawnerEntity, world.Transforms, world.Parents)
	worldRot, _ := tm.GetWorldRotation(spawnerEntity, world.Transforms, world.Parents)

	spawnerOffset, err := sm.GetOffset(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting offset of spawner entity %d: %v", spawnerEntity, err)
	}

	sType, err := sm.GetSpawnerType(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting spawner type of spawner entity %d: %v", spawnerEntity, err)
	}

	shape, err := sm.GetShape(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting shape of spawner entity %d: %v", spawnerEntity, err)
	}

	var finalOffset utils.Vec2
	switch sType {
	case ecs.SpawnerType_Point:
		cos := math.Cos(worldRot)
		sin := math.Sin(worldRot)

		rotatedOffset := utils.Vec2{
			X: (spawnerOffset.X*cos - spawnerOffset.Y*sin),
			Y: (spawnerOffset.X*sin + spawnerOffset.Y*cos),
		}

		finalOffset = rotatedOffset
	case ecs.SpawnerType_Inside:
		finalOffset = shape.GetRandomPoint(world.Rng).Add(worldPos).Add(spawnerOffset)
	case ecs.SpawnerType_Perimeter:
		finalOffset = shape.GetRandomPointAroundShape(world.Rng).Add(worldPos).Add(spawnerOffset)
	}

	err = tm.SetWorldPos(newEntity, worldPos.Add(finalOffset), world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world position of new entity %d: %v", newEntity, err)
	}

	err = tm.SetWorldRotation(newEntity, worldRot, world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world rotation of new entity %d: %v", newEntity, err)
	}

	return nil
}
