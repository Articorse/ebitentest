package data

const (
	// Under the entities' sprites, since a site is part of the ground until it's built
	ConstructionSiteSpriteLayer = 10
	// A blueprint is drawn in shades of this blue, from the brightness of the finished tile, so that it stands out from
	// the ground whatever it's made of but still shows what it is
	BlueprintR, BlueprintG, BlueprintB = 0.3, 0.6, 1
	// Red for one about to be put down where it can't go
	InvalidBlueprintR, InvalidBlueprintG, InvalidBlueprintB = 1, 0.3, 0.3
	// The shade of the tile's darkest parts, and how much brighter its brightest get
	BlueprintShadeBase, BlueprintShadeRange = 0.4, 1.2
	BlueprintAlpha                          = 0.5

	PlayerBuildRangeTiles = 3
	// A wooden wall takes 10 work, so 10 strikes
	PlayerBuildWork       = 1
	PlayerBuildCooldownMs = 250
)
