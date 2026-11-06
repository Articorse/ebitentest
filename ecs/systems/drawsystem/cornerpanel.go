package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// The panel of slots in the bottom right corner that build mode and the area menu choose from, data.CornerPanelColumns
// to a row
type cornerLayout struct {
	slotSize utils.Vec2f
	panel    image.Rectangle
	start    utils.Vec2f
}

func newCornerLayout(screenSize utils.Vec2f, count int, slotSize utils.Vec2f) cornerLayout {
	cols := min(max(count, 1), data.CornerPanelColumns)
	rows := max(1, (count+data.CornerPanelColumns-1)/data.CornerPanelColumns)
	gridW := float64(cols)*slotSize.X + float64(cols-1)*data.HotbarSlotGapPx
	gridH := float64(rows)*slotSize.Y + float64(rows-1)*data.HotbarSlotGapPx
	pad := float64(data.InventoryPanelPaddingPx)
	start := utils.Vec2f{X: screenSize.X - data.InventoryMarginPx - pad - gridW, Y: screenSize.Y - data.InventoryMarginPx - pad - gridH}
	panel := image.Rect(int(start.X-pad), int(start.Y-pad), int(start.X+gridW+pad), int(start.Y+gridH+pad))
	return cornerLayout{slotSize: slotSize, panel: panel, start: start}
}

func (l cornerLayout) slotPos(i int) utils.Vec2f {
	col, row := i%data.CornerPanelColumns, i/data.CornerPanelColumns
	return utils.Vec2f{
		X: l.start.X + float64(col)*(l.slotSize.X+data.HotbarSlotGapPx),
		Y: l.start.Y + float64(row)*(l.slotSize.Y+data.HotbarSlotGapPx),
	}
}

func (l cornerLayout) slotAt(cursor utils.Vec2f, count int) int {
	for i := range count {
		if within(cursor, l.slotPos(i), l.slotSize) {
			return i
		}
	}
	return -1
}

func (l cornerLayout) covers(cursor utils.Vec2f) bool {
	return within(cursor, utils.Vec2f{X: float64(l.panel.Min.X), Y: float64(l.panel.Min.Y)},
		utils.Vec2f{X: float64(l.panel.Dx()), Y: float64(l.panel.Dy())})
}

// The panel and its slots, the chosen one highlighted, with icon drawing what goes in each at its top left corner
func drawCornerPanel(screen *ebiten.Image, scale float64, l cornerLayout, count, chosen int, assets *assetmanager.AssetManager, icon func(i int, p utils.Vec2f) error) error {
	if err := drawNineSlice(screen, scale, l.panel, assets); err != nil {
		return err
	}
	slot, err := assets.GetAsset(common.AssetImageSlot)
	if err != nil {
		return err
	}
	highlight, err := assets.GetAsset(common.AssetImageSlotHighlight)
	if err != nil {
		return err
	}
	for i := range count {
		p := l.slotPos(i)
		bg := slot.Frames[0]
		if i == chosen {
			bg = highlight.Frames[0]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(p.X, p.Y)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(bg, op)
		if err := icon(i, p); err != nil {
			return err
		}
	}
	return nil
}
