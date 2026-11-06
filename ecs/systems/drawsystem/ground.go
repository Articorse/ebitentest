package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"strconv"
)

// What lies on the tiles in view, each on its tile as its item looks
func GroundItemSprites(bounds [2]utils.Vec2f, ecsContainer *ecs.ECSContainer) []common.StaticSprite {
	first, last := utils.WorldPosToTilePos(bounds[0]), utils.WorldPosToTilePos(bounds[1])
	sprites := []common.StaticSprite{}
	for _, t := range ecsContainer.Terrain().GroundStacksIn(first, last) {
		stack, _ := ecsContainer.Terrain().GroundStack(t)
		image, ok := ecsContainer.GroundStackImage(stack)
		if !ok {
			continue
		}
		sprites = append(sprites, common.StaticSprite{ImageAsset: image, Pos: utils.TilePosToWorldPos(t), Layer: data.ItemSpriteLayer})
	}
	return sprites
}

// Its name, with how many there are of a stack of more than one
func GroundStackTooltip(stack ecs.GroundStack, ecsContainer *ecs.ECSContainer) []string {
	name := ecsContainer.GroundStackName(stack)
	if stack.Unique == nil && stack.Count > 1 {
		return []string{strconv.Itoa(stack.Count) + " " + name}
	}
	return []string{name}
}
