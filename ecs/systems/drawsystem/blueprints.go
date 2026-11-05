package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The blueprint chosen in the inventory menu, to be put down with a click. Like the menu, it only changes what's drawn
// and what a click sends.
type BlueprintPlacer struct {
	kind data.BlueprintEnum
}

func (p *BlueprintPlacer) Start(kind data.BlueprintEnum) {
	p.kind = kind
}

func (p *BlueprintPlacer) Stop() {
	p.kind = data.Blueprint_None
}

func (p *BlueprintPlacer) Kind() (data.BlueprintEnum, bool) {
	return p.kind, p.kind != data.Blueprint_None
}

// The finished tile as a blueprint on the tile: blue if it can go there and red if it can't
func (p *BlueprintPlacer) Draw(
	dst *ebiten.Image,
	camera utils.Vec2f,
	tile utils.Vec2i,
	valid bool,
	atlas *ecs.BlueprintAtlas,
	assets *assetmanager.AssetManager,
) error {
	def, ok := atlas.Def(p.kind)
	if !ok {
		return nil
	}
	asset, err := assets.GetAsset(def.ImageAsset())
	if err != nil {
		return fmt.Errorf("error getting the image of blueprint %d: %w", p.kind, err)
	}

	img := asset.Frames[0]
	pos := utils.TilePosToWorldPos(tile)
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(pos.X-camera.X-float64(img.Bounds().Dx())/2, pos.Y-camera.Y-float64(img.Bounds().Dy())/2)
	colorm.DrawImage(dst, img, blueprintColorM(valid), op)
	return nil
}

// Its name, then a line for each of its costs
func BlueprintTooltip(def ecs.BlueprintDef, items *ecs.ItemAtlas) []string {
	lines := []string{def.Name()}
	for _, c := range def.Costs() {
		name := "?"
		if item, ok := items.Def(c.Kind); ok {
			name = item.Name()
		}
		lines = append(lines, strconv.Itoa(c.Count)+" "+name)
	}
	return lines
}

// Its name, how much of each cost has been delivered, and how far building it has come, from 0% when it's placed
func SiteTooltip(site ecs.ConstructionSite, def ecs.BlueprintDef, items *ecs.ItemAtlas) []string {
	lines := []string{def.Name()}
	for _, c := range def.Costs() {
		name := "?"
		if item, ok := items.Def(c.Kind); ok {
			name = item.Name()
		}
		delivered := c.Count - site.Needs(c.Kind, def)
		lines = append(lines, fmt.Sprintf("%s %d/%d", name, delivered, c.Count))
	}
	progress := 0
	if site.MaxHealth > 1 {
		progress = (site.Health - 1) * 100 / (site.MaxHealth - 1)
	}
	return append(lines, fmt.Sprintf("Construction %d%%", progress))
}

// Below and to the right of the cursor, in UI pixels, moved back onto the screen where it would go off it
func DrawTooltip(screen *ebiten.Image, scale float64, cursor utils.Vec2f, lines []string, assets *assetmanager.AssetManager) error {
	if len(lines) == 0 {
		return nil
	}
	face, err := floatingTextFace(data.TooltipTextSizePx * scale)
	if err != nil {
		return fmt.Errorf("error getting the font for tooltips: %w", err)
	}

	width, lineHeight := 0.0, 0.0
	for _, l := range lines {
		w, h := text.Measure(l, face, 0)
		width, lineHeight = max(width, w/scale), max(lineHeight, h/scale)
	}
	pad := float64(data.TooltipPaddingPx)
	size := utils.Vec2f{X: width + 2*pad, Y: lineHeight*float64(len(lines)) + 2*pad}

	screenSize := ImageSize(screen).Multiply(1 / scale)
	topLeft := cursor.Add(utils.Vec2f{X: data.TooltipOffsetPx, Y: data.TooltipOffsetPx})
	topLeft.X = max(0, min(topLeft.X, screenSize.X-size.X))
	topLeft.Y = max(0, min(topLeft.Y, screenSize.Y-size.Y))

	panel := image.Rect(int(topLeft.X), int(topLeft.Y), int(math.Ceil(topLeft.X+size.X)), int(math.Ceil(topLeft.Y+size.Y)))
	if err := drawNineSlice(screen, scale, panel, assets); err != nil {
		return err
	}
	for i, l := range lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate((float64(panel.Min.X)+pad)*scale, (float64(panel.Min.Y)+pad+float64(i)*lineHeight)*scale)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, l, face, op)
	}
	return nil
}

// The panel of blueprints in the bottom right corner of the screen, in UI pixels
type blueprintLayout struct {
	slotSize utils.Vec2f
	panel    image.Rectangle
	start    utils.Vec2f
}

func newBlueprintLayout(screenSize utils.Vec2f, count int, slotSize utils.Vec2f) blueprintLayout {
	cols := min(max(count, 1), data.BlueprintColumns)
	rows := max(1, (count+data.BlueprintColumns-1)/data.BlueprintColumns)
	gridW := float64(cols)*slotSize.X + float64(cols-1)*data.HotbarSlotGapPx
	gridH := float64(rows)*slotSize.Y + float64(rows-1)*data.HotbarSlotGapPx
	pad := float64(data.InventoryPanelPaddingPx)
	start := utils.Vec2f{X: screenSize.X - data.InventoryMarginPx - pad - gridW, Y: screenSize.Y - data.InventoryMarginPx - pad - gridH}
	panel := image.Rect(int(start.X-pad), int(start.Y-pad), int(start.X+gridW+pad), int(start.Y+gridH+pad))
	return blueprintLayout{slotSize: slotSize, panel: panel, start: start}
}

func (l blueprintLayout) slotPos(i int) utils.Vec2f {
	col, row := i%data.BlueprintColumns, i/data.BlueprintColumns
	return utils.Vec2f{
		X: l.start.X + float64(col)*(l.slotSize.X+data.HotbarSlotGapPx),
		Y: l.start.Y + float64(row)*(l.slotSize.Y+data.HotbarSlotGapPx),
	}
}

func (l blueprintLayout) slotAt(cursor utils.Vec2f, count int) int {
	for i := range count {
		if within(cursor, l.slotPos(i), l.slotSize) {
			return i
		}
	}
	return -1
}

func (l blueprintLayout) covers(cursor utils.Vec2f) bool {
	return within(cursor, utils.Vec2f{X: float64(l.panel.Min.X), Y: float64(l.panel.Min.Y)},
		utils.Vec2f{X: float64(l.panel.Dx()), Y: float64(l.panel.Dy())})
}

func drawBlueprintPanel(screen *ebiten.Image, scale float64, l blueprintLayout, atlas *ecs.BlueprintAtlas, slot *assetmanager.ImageAsset, assets *assetmanager.AssetManager) error {
	if err := drawNineSlice(screen, scale, l.panel, assets); err != nil {
		return err
	}
	for i, kind := range atlas.Kinds() {
		p := l.slotPos(i)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(p.X, p.Y)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(slot.Frames[0], op)

		def, _ := atlas.Def(kind)
		icon, err := assets.GetAsset(def.ImageAsset())
		if err != nil {
			return fmt.Errorf("error getting the image of blueprint %d: %w", kind, err)
		}
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(p.X+(l.slotSize.X-icon.FrameSize.X)/2, p.Y+(l.slotSize.Y-icon.FrameSize.Y)/2)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(icon.Frames[0], op)
	}
	return nil
}
