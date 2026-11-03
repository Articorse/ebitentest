package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// One slot per inventory slot, in a column centered on the left edge of the screen, with each item drawn as the first
// frame of its sprite
// Laid out in UI pixels, which are the scale's number of screen pixels across
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

	slotW, slotH := slot.FrameSize.X, slot.FrameSize.Y
	total := float64(inv.Capacity())*slotH + float64(inv.Capacity()-1)*data.HotbarSlotGapPx
	top := (float64(screen.Bounds().Dy())/scale - total) / 2

	for i := range inv.Capacity() {
		x, y := float64(data.HotbarMarginPx), top+float64(i)*(slotH+data.HotbarSlotGapPx)

		bg := slot.Frames[0]
		if i == inv.Selected() {
			bg = highlight.Frames[0]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(bg, op)

		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		spr, ok := ecsContainer.Sprites.Get(item)
		if !ok {
			continue
		}
		icon, err := assetManager.GetAsset(spr.ImageAsset())
		if err != nil {
			return fmt.Errorf("error getting icon of item %d: %w", item, err)
		}

		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x+(slotW-icon.FrameSize.X)/2, y+(slotH-icon.FrameSize.Y)/2)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(icon.Frames[0], op)

		if itemComp, ok := ecsContainer.Items.Get(item); ok && itemComp.Count() > 1 {
			if err := drawStackCount(screen, scale, itemComp.Count(), x+slotW, y+slotH); err != nil {
				return err
			}
		}
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
