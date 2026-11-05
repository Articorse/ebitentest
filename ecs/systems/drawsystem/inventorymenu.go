package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The player's inventory, as a grid of slots at the bottom of the screen, below the player. An item is picked up with a
// click and put down with another; nothing in the simulation changes until it's put down, so closing the menu while
// holding an item leaves it where it was. Positions are in UI pixels throughout.
type InventoryMenu struct {
	open bool
	// The inventory slot of the item picked up
	held    int
	holding bool
}

func (m *InventoryMenu) IsOpen() bool {
	return m.open
}

func (m *InventoryMenu) Toggle() {
	m.open = !m.open
	m.holding = false
}

type inventoryLayout struct {
	slotSize  utils.Vec2f
	panel     image.Rectangle
	gridStart utils.Vec2f
	screenH   float64
}

func newInventoryLayout(screenSize utils.Vec2f, assets *assetmanager.AssetManager) (inventoryLayout, error) {
	slot, err := assets.GetAsset(common.AssetImageSlot)
	if err != nil {
		return inventoryLayout{}, err
	}
	s := slot.FrameSize
	gridW := data.InventoryColumns*s.X + (data.InventoryColumns-1)*data.HotbarSlotGapPx
	gridH := data.InventoryRows*s.Y + (data.InventoryRows-1)*data.HotbarSlotGapPx
	pad := float64(data.InventoryPanelPaddingPx)
	start := utils.Vec2f{X: (screenSize.X - gridW) / 2, Y: screenSize.Y - data.InventoryMarginPx - pad - gridH}
	panel := image.Rect(int(start.X-pad), int(start.Y-pad), int(start.X+gridW+pad), int(start.Y+gridH+pad))
	return inventoryLayout{slotSize: s, panel: panel, gridStart: start, screenH: screenSize.Y}, nil
}

func (l inventoryLayout) slotPos(i int) utils.Vec2f {
	col, row := i%data.InventoryColumns, i/data.InventoryColumns
	return utils.Vec2f{
		X: l.gridStart.X + float64(col)*(l.slotSize.X+data.HotbarSlotGapPx),
		Y: l.gridStart.Y + float64(row)*(l.slotSize.Y+data.HotbarSlotGapPx),
	}
}

// The held item, unless its slot was emptied since, such as by moving all of it onto a stack
func (m *InventoryMenu) heldItem(inv *ecs.Inventory) (common.EntityId, bool) {
	if !m.holding {
		return -1, false
	}
	return inv.Slot(m.held)
}

func within(p, topLeft, size utils.Vec2f) bool {
	return p.X >= topLeft.X && p.X < topLeft.X+size.X && p.Y >= topLeft.Y && p.Y < topLeft.Y+size.Y
}

// -1 if the cursor isn't on one
func (l inventoryLayout) slotAt(cursor utils.Vec2f, slots int) int {
	for i := range slots {
		if within(cursor, l.slotPos(i), l.slotSize) {
			return i
		}
	}
	return -1
}

func (l inventoryLayout) hotbarSlotAt(cursor utils.Vec2f, slots int) int {
	for i := range slots {
		x, y := hotbarSlotPos(i, slots, l.screenH, l.slotSize)
		if within(cursor, utils.Vec2f{X: x, Y: y}, l.slotSize) {
			return i
		}
	}
	return -1
}

// Whether clicks at the cursor are the menu's rather than the world's: over its panel or the hotbar while it's open,
// and anywhere while an item is held
func (m *InventoryMenu) Covers(cursor, screenSize utils.Vec2f, inv *ecs.Inventory, assets *assetmanager.AssetManager) bool {
	if !m.open {
		return false
	}
	l, err := newInventoryLayout(screenSize, assets)
	if err != nil {
		return false
	}
	overPanel := cursor.X >= float64(l.panel.Min.X) && cursor.X < float64(l.panel.Max.X) &&
		cursor.Y >= float64(l.panel.Min.Y) && cursor.Y < float64(l.panel.Max.Y)
	_, holding := m.heldItem(inv)
	return holding || overPanel || l.hotbarSlotAt(cursor, inv.HotbarSize()) != -1
}

// The inventory slot under the cursor, or -1
func (m *InventoryMenu) SlotAt(cursor, screenSize utils.Vec2f, inv *ecs.Inventory, assets *assetmanager.AssetManager) int {
	if !m.open {
		return -1
	}
	l, err := newInventoryLayout(screenSize, assets)
	if err != nil {
		return -1
	}
	return l.slotAt(cursor, inv.Capacity())
}

// Points the hotbar slot at the item under the cursor, or at nothing if the slot under it is empty
func (m *InventoryMenu) HotbarKey(hotbarSlot int, cursor, screenSize utils.Vec2f, inv *ecs.Inventory, assets *assetmanager.AssetManager) ecs.InventoryAction {
	slot := m.SlotAt(cursor, screenSize, inv, assets)
	if slot == -1 {
		return ecs.InventoryAction{}
	}
	if _, full := inv.Slot(slot); full {
		return ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: slot, To: hotbarSlot}
	}
	return ecs.InventoryAction{Kind: ecs.InventoryAction_Unbind, To: hotbarSlot}
}

// What a click at the cursor does. Picking up and putting back only change the menu; putting an item down elsewhere
// moves it, onto a hotbar slot points that slot at it, and clicking a hotbar slot with nothing held points it at
// nothing. Putting an item down on another picks that one up, and onto a stack of its kind keeps what didn't fit: both
// end up in the slot the held item came from, so the menu goes on holding that slot, and lets go if it's emptied.
func (m *InventoryMenu) Click(cursor, screenSize utils.Vec2f, inv *ecs.Inventory, assets *assetmanager.AssetManager) ecs.InventoryAction {
	if !m.open {
		return ecs.InventoryAction{}
	}
	l, err := newInventoryLayout(screenSize, assets)
	if err != nil {
		return ecs.InventoryAction{}
	}
	if _, ok := m.heldItem(inv); !ok {
		m.holding = false
	}

	if slot := l.slotAt(cursor, inv.Capacity()); slot != -1 {
		switch _, full := inv.Slot(slot); {
		case !m.holding && full:
			m.held, m.holding = slot, true
		case m.holding && slot == m.held:
			m.holding = false
		case m.holding:
			m.holding = full
			return ecs.InventoryAction{Kind: ecs.InventoryAction_Move, From: m.held, To: slot}
		}
		return ecs.InventoryAction{}
	}

	if h := l.hotbarSlotAt(cursor, inv.HotbarSize()); h != -1 {
		if m.holding {
			m.holding = false
			return ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: m.held, To: h}
		}
		if _, bound := inv.HotbarItem(h); bound {
			return ecs.InventoryAction{Kind: ecs.InventoryAction_Unbind, To: h}
		}
	}
	return ecs.InventoryAction{}
}

// The panel and its slots, each item with the number of the hotbar slot pointing at it, and the held item under the
// cursor
func (m *InventoryMenu) Draw(
	screen *ebiten.Image,
	scale float64,
	cursor utils.Vec2f,
	owner common.EntityId,
	ecsContainer *ecs.ECSContainer,
	assets *assetmanager.AssetManager,
) error {
	inv, ok := ecsContainer.Inventories.Get(owner)
	if !m.open || !ok {
		return nil
	}
	l, err := newInventoryLayout(ImageSize(screen).Multiply(1/scale), assets)
	if err != nil {
		return err
	}
	if err := drawNineSlice(screen, scale, l.panel, assets); err != nil {
		return err
	}

	slot, err := assets.GetAsset(common.AssetImageSlot)
	if err != nil {
		return err
	}
	for i := range inv.Capacity() {
		p := l.slotPos(i)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(p.X, p.Y)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(slot.Frames[0], op)

		item, ok := inv.Slot(i)
		if held, holding := m.heldItem(inv); !ok || (holding && item == held) {
			continue
		}
		if err := drawSlotItem(screen, scale, item, p.X+l.slotSize.X/2, p.Y+l.slotSize.Y/2, l.slotSize, ecsContainer, assets); err != nil {
			return err
		}
		if h := inv.HotbarSlotOf(item); h != -1 {
			if err := drawHotbarNumber(screen, scale, h+1, p.X, p.Y); err != nil {
				return err
			}
		}
	}

	if item, ok := m.heldItem(inv); ok {
		return drawSlotItem(screen, scale, item, cursor.X, cursor.Y, l.slotSize, ecsContainer, assets)
	}
	return nil
}

// In the top left corner of a slot, whose top left is at x, y in UI pixels
func drawHotbarNumber(screen *ebiten.Image, scale float64, number int, x, y float64) error {
	face, err := floatingTextFace(data.HotbarStackCountSizePx * scale)
	if err != nil {
		return fmt.Errorf("error getting the font for hotbar numbers: %w", err)
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate((x+data.HotbarStackCountInsetPx)*scale, (y+data.HotbarStackCountInsetPx)*scale)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, strconv.Itoa(number), face, op)
	return nil
}

// The panel image stretched over the rectangle, in UI pixels, keeping its corners as they are
func drawNineSlice(screen *ebiten.Image, scale float64, r image.Rectangle, assets *assetmanager.AssetManager) error {
	panel, err := assets.GetAsset(common.AssetImageInventoryPanel)
	if err != nil {
		return err
	}
	img := panel.Frames[0]
	const s = data.InventoryPanelSlicePx
	srcEdges := [4]int{0, s, img.Bounds().Dx() - s, img.Bounds().Dx()}
	srcEdgesY := [4]int{0, s, img.Bounds().Dy() - s, img.Bounds().Dy()}
	dstX := [4]int{r.Min.X, r.Min.X + s, r.Max.X - s, r.Max.X}
	dstY := [4]int{r.Min.Y, r.Min.Y + s, r.Max.Y - s, r.Max.Y}

	for row := range 3 {
		for col := range 3 {
			src := image.Rect(srcEdges[col], srcEdgesY[row], srcEdges[col+1], srcEdgesY[row+1])
			w, h := dstX[col+1]-dstX[col], dstY[row+1]-dstY[row]
			if src.Dx() == 0 || src.Dy() == 0 || w <= 0 || h <= 0 {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(w)/float64(src.Dx()), float64(h)/float64(src.Dy()))
			op.GeoM.Translate(float64(dstX[col]), float64(dstY[row]))
			op.GeoM.Scale(scale, scale)
			screen.DrawImage(img.SubImage(src.Add(img.Bounds().Min)).(*ebiten.Image), op)
		}
	}
	return nil
}
