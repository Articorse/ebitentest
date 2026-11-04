package common

import "ebittest/utils"

// An image drawn among the entities' sprites that isn't an entity, such as a tree
type StaticSprite struct {
	ImageAsset ImageAssetTag
	Frame      int
	// Where the image's center is
	Pos   utils.Vec2f
	Layer uint8
	// Drawn see-through and tinted, as what's planned rather than what's there
	Ghost bool
}
