package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"strconv"
)

// What's in the storage areas in view, each lying on the tile its slot is for, as the item looks lying loose
func StoredItemSprites(bounds [2]utils.Vec2f, ecsContainer *ecs.ECSContainer) []common.StaticSprite {
	first, last := utils.WorldPosToTilePos(bounds[0]), utils.WorldPosToTilePos(bounds[1])
	sprites := []common.StaticSprite{}
	for e, area := range ecs.Query1(&ecsContainer.StorageAreas) {
		inv, ok := ecsContainer.Inventories.Get(e)
		if !ok {
			continue
		}
		for i := range inv.Capacity() {
			t := area.SlotTile(i)
			if t.X < first.X || t.X > last.X || t.Y < first.Y || t.Y > last.Y {
				continue
			}
			item, ok := inv.Slot(i)
			if !ok {
				continue
			}
			spr, ok := ecsContainer.Sprites.Get(item)
			if !ok {
				continue
			}
			sprites = append(sprites, common.StaticSprite{
				ImageAsset: spr.ImageAsset(),
				Frame:      spr.SubImageIdx(),
				Pos:        utils.TilePosToWorldPos(t),
				Layer:      data.ItemSpriteLayer,
			})
		}
	}
	return sprites
}

// A line for how many there are of each kind of item in it, under its name
func StorageTooltip(contents []ecs.ItemCost, items *ecs.ItemAtlas) []string {
	lines := []string{"Storage"}
	for _, c := range contents {
		name := "?"
		if item, ok := items.Def(c.Kind); ok {
			name = item.Name()
		}
		lines = append(lines, strconv.Itoa(c.Count)+" "+name)
	}
	if len(contents) == 0 {
		lines = append(lines, "Empty")
	}
	return lines
}
