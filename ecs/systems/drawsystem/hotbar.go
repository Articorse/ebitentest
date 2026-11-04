package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// One slot per hotbar slot, in a column centered on the left edge of the screen, with the item each points at drawn as
// the first frame of its sprite. Laid out in UI pixels, which are the scale's number of screen pixels across.
func DrawHotbar(
	screen *ebiten.Image,
	scale float64,
	owner common.EntityId,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	inv, ok := ecsContainer.Inventories.Get(owner)
	if !ok {
		return nil
	}

	slot, err := assetManager.GetAsset(common.AssetImageSlot)
	if err != nil {
		return err
	}
	highlight, err := assetManager.GetAsset(common.AssetImageSlotHighlight)
	if err != nil {
		return err
	}

	for i := range inv.HotbarSize() {
		x, y := hotbarSlotPos(i, inv.HotbarSize(), ImageSize(screen).Y/scale, slot.FrameSize)

		bg := slot.Frames[0]
		if i == inv.Selected() {
			bg = highlight.Frames[0]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(bg, op)

		if item, ok := inv.HotbarItem(i); ok {
			if err := drawSlotItem(screen, scale, item, x+slot.FrameSize.X/2, y+slot.FrameSize.Y/2, slot.FrameSize, ecsContainer, assetManager); err != nil {
				return err
			}
		}
	}

	return nil
}

// The top left corner of the hotbar slot, in UI pixels, on a screen of the given height in UI pixels
func hotbarSlotPos(i, slots int, screenHeight float64, slotSize utils.Vec2f) (x, y float64) {
	total := float64(slots)*slotSize.Y + float64(slots-1)*data.HotbarSlotGapPx
	top := (screenHeight - total) / 2
	return data.HotbarMarginPx, top + float64(i)*(slotSize.Y+data.HotbarSlotGapPx)
}

// The item's icon, the first frame of its sprite, centered on x, y in UI pixels, and its count if it's a stack, in the
// bottom right corner of a slot of the given size
func drawSlotItem(
	screen *ebiten.Image,
	scale float64,
	item common.EntityId,
	x, y float64,
	slotSize utils.Vec2f,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	spr, ok := ecsContainer.Sprites.Get(item)
	if !ok {
		return nil
	}
	icon, err := assetManager.GetAsset(spr.ImageAsset())
	if err != nil {
		return fmt.Errorf("error getting icon of item %d: %w", item, err)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-icon.FrameSize.X/2, y-icon.FrameSize.Y/2)
	op.GeoM.Scale(scale, scale)
	screen.DrawImage(icon.Frames[0], op)

	if itemComp, ok := ecsContainer.Items.Get(item); ok && itemComp.Count() > 1 {
		return drawStackCount(screen, scale, itemComp.Count(), x+slotSize.X/2, y+slotSize.Y/2)
	}
	return nil
}

// In the bottom right corner of a slot, whose bottom right is at x, y in UI pixels. Drawn at the size it takes up on
// screen rather than scaled up, so that it's crisp.
func drawStackCount(screen *ebiten.Image, scale float64, count int, x, y float64) error {
	face, err := floatingTextFace(data.HotbarStackCountSizePx * scale)
	if err != nil {
		return fmt.Errorf("error getting the font for stack counts: %w", err)
	}

	label := strconv.Itoa(count)
	w, h := text.Measure(label, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate((x-data.HotbarStackCountInsetPx)*scale-w, (y-data.HotbarStackCountInsetPx)*scale-h)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, label, face, op)

	return nil
}
