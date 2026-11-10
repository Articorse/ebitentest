package data

const (
	ItemSpriteLayer = 20
	// How far around where an item's dropped it can end up when there's no room for it there
	GroundSpillRangeTiles = 8
	// How far away what lies on the ground can be picked up, or something put down, which is as far as building reaches
	ItemReachTiles = PlayerBuildRangeTiles
	// High enough that it rarely holds the player back, unlike a colonist
	PlayerMaxCarryMg = 250 * MilligramsPerKg
)
