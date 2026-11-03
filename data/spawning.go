package data

const (
	// How often spawning an enemy is tried, and how many random tiles each try picks before giving up
	EnemySpawnIntervalTicks = TPS
	EnemySpawnPicks         = 20
	// Spawned enemies appear this far from the player, out of its sight, and are removed once they're further than the
	// greatest distance
	EnemySpawnMinRangeTiles = 64
	EnemySpawnMaxRangeTiles = 128
	// Below the night sky's light, so that enemies only spawn in the dark, such as deep in caves
	EnemySpawnMaxLight = 40
	MaxSpawnedEnemies  = 20
)
