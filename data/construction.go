package data

const (
	// Under the entities' sprites, since a site is part of the ground until it's built
	ConstructionSiteSpriteLayer = 10
	// Light blue and half see-through, for what's planned rather than built
	GhostTintR, GhostTintG, GhostTintB = 0.6, 0.85, 1
	GhostAlpha                         = 0.5
	// The ghost of a blueprint about to be put down where it can't go
	InvalidGhostTintR, InvalidGhostTintG, InvalidGhostTintB = 1, 0.35, 0.35
)
