package enemyspawning

import (
	"ebittest/content/templates"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

// Removes spawned enemies that are too far from the player, then, every data.EnemySpawnIntervalTicks and while there are
// fewer than data.MaxSpawnedEnemies, spawns an evil slime on the first of a few random tiles that qualifies. Removed
// rather than killed, so that leaving them behind doesn't drop anything.
func Tick(ecsContainer *ecs.ECSContainer, player common.EntityId) error {
	playerPos, err := ecsContainer.WorldPos(player)
	if err != nil {
		return fmt.Errorf("error getting the player's position for spawning enemies: %w", err)
	}
	center := utils.WorldPosToTilePos(playerPos)

	remaining := 0
	for e := range ecs.Query1(&ecsContainer.SpawnedEnemies) {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			continue
		}
		if utils.Dist2(center, utils.WorldPosToTilePos(pos)) > data.EnemySpawnMaxRangeTiles*data.EnemySpawnMaxRangeTiles {
			ecsContainer.ScheduleRemoveEntity(e)
			continue
		}
		remaining++
	}

	if ecsContainer.TickIdx%data.EnemySpawnIntervalTicks != 0 || remaining >= data.MaxSpawnedEnemies {
		return nil
	}

	var vision *ecs.Vision
	if v, ok := ecsContainer.Visions.Get(player); ok {
		vision = v
	}
	light := &ecs.LightMap{}
	const side = 2*data.EnemySpawnMaxRangeTiles + 1
	for range data.EnemySpawnPicks {
		t := utils.Vec2i{
			X: center.X + ecsContainer.Rng.IntN(side) - data.EnemySpawnMaxRangeTiles,
			Y: center.Y + ecsContainer.Rng.IntN(side) - data.EnemySpawnMaxRangeTiles,
		}
		if !canSpawnAt(t, center, vision, light, ecsContainer) {
			continue
		}

		slime, err := templates.SpawnEvilSlime(ecsContainer, utils.TilePosToWorldPos(t), player)
		if err != nil {
			return err
		}
		ecsContainer.AddComponent(slime, ecs.NewSpawnedEnemyComponent())
		return nil
	}
	return nil
}

// Within the spawning ring around the player, out of its sight, dark, and free walkable ground
func canSpawnAt(t, center utils.Vec2i, vision *ecs.Vision, light *ecs.LightMap, ecsContainer *ecs.ECSContainer) bool {
	d := utils.Dist2(center, t)
	if d < data.EnemySpawnMinRangeTiles*data.EnemySpawnMinRangeTiles || d > data.EnemySpawnMaxRangeTiles*data.EnemySpawnMaxRangeTiles {
		return false
	}
	if vision != nil && vision.CanSee(t) {
		return false
	}
	if ecsContainer.Terrain().IsBlockedAt(utils.TilePosToWorldPos(t)) || ecsContainer.IsTileOccupied(t) {
		return false
	}

	ecsContainer.LightAround(t, 0, light)
	return light.At(t) < data.EnemySpawnMaxLight
}
