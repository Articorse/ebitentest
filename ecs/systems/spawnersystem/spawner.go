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

	newEntity := world.AddEntity()

	for _, comp := range comps {
		world.AddComponent(newEntity, comp)
	}

	worldPos, err := tm.GetWorldPos(spawnerEntity, world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error getting world position of spawner entity %d: %v", spawnerEntity, err)
	}

	worldRot, err := tm.GetWorldRotation(spawnerEntity, world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of spawner entity %d: %v", spawnerEntity, err)
	}

	spawnerOffset, err := sm.GetOffset(spawnerEntity, world.Spawners)
	if err != nil {
		return fmt.Errorf("error getting offset of spawner entity %d: %v", spawnerEntity, err)
	}

	cos := math.Cos(worldRot)
	sin := math.Sin(worldRot)

	rotatedOffset := utils.Vec2{
		X: (spawnerOffset.X*cos - spawnerOffset.Y*sin),
		Y: (spawnerOffset.X*sin + spawnerOffset.Y*cos),
	}

	err = tm.SetWorldPos(newEntity, worldPos.Add(rotatedOffset), world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world position of new entity %d: %v", newEntity, err)
	}

	err = tm.SetWorldRotation(newEntity, worldRot, world.Transforms, world.Parents)
	if err != nil {
		return fmt.Errorf("error setting world rotation of new entity %d: %v", newEntity, err)
	}

	return nil
}
