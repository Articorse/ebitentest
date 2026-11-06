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
	"math"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Build mode: the blueprint chosen, put down with a click or along the outline of a dragged rectangle, or storage,
// marked out by dragging rectangles, and the panel to choose from in the bottom right corner while it's on, with
// storage after the blueprints. Like the inventory
// menu, it only changes what's drawn and what a click sends. Positions on the panel are in UI pixels.
type BlueprintPlacer struct {
	kind     data.BlueprintEnum
	storage  bool
	dragging bool
	start    utils.Vec2i
}

func (p *BlueprintPlacer) Start(kind data.BlueprintEnum) {
	p.kind, p.storage, p.dragging = kind, false, false
}

func (p *BlueprintPlacer) StartStorage() {
	p.kind, p.storage, p.dragging = data.Blueprint_None, true, false
}

// Starts with the first blueprint chosen
func (p *BlueprintPlacer) Toggle(atlas *ecs.BlueprintAtlas) {
	if p.IsActive() {
		p.Stop()
		return
	}
	if kinds := atlas.Kinds(); len(kinds) > 0 {
		p.Start(kinds[0])
	} else {
		p.StartStorage()
	}
}

func (p *BlueprintPlacer) Stop() {
	p.kind, p.storage, p.dragging = data.Blueprint_None, false, false
}

func (p *BlueprintPlacer) IsActive() bool {
	return p.kind != data.Blueprint_None || p.storage
}

// The blueprint chosen, if it's a blueprint rather than storage
func (p *BlueprintPlacer) Kind() (data.BlueprintEnum, bool) {
	return p.kind, p.kind != data.Blueprint_None
}

func (p *BlueprintPlacer) IsStorage() bool {
	return p.storage
}

func (p *BlueprintPlacer) layout(screenSize utils.Vec2f, atlas *ecs.BlueprintAtlas, assets *assetmanager.AssetManager) (cornerLayout, bool) {
	slot, err := assets.GetAsset(common.AssetImageSlot)
	if !p.IsActive() || err != nil {
		return cornerLayout{}, false
	}
	return newCornerLayout(screenSize, len(atlas.Kinds())+1, slot.FrameSize), true
}

// Whether clicks at the cursor are the panel's rather than the world's
func (p *BlueprintPlacer) Covers(cursor, screenSize utils.Vec2f, atlas *ecs.BlueprintAtlas, assets *assetmanager.AssetManager) bool {
	l, ok := p.layout(screenSize, atlas, assets)
	return ok && l.covers(cursor)
}

// The blueprint under the cursor on the panel
func (p *BlueprintPlacer) BlueprintAt(cursor, screenSize utils.Vec2f, atlas *ecs.BlueprintAtlas, assets *assetmanager.AssetManager) (data.BlueprintEnum, bool) {
	l, ok := p.layout(screenSize, atlas, assets)
	if !ok {
		return data.Blueprint_None, false
	}
	kinds := atlas.Kinds()
	if i := l.slotAt(cursor, len(kinds)); i != -1 {
		return kinds[i], true
	}
	return data.Blueprint_None, false
}

// Whether the cursor is on the panel's storage button
func (p *BlueprintPlacer) StorageAt(cursor, screenSize utils.Vec2f, atlas *ecs.BlueprintAtlas, assets *assetmanager.AssetManager) bool {
	l, ok := p.layout(screenSize, atlas, assets)
	kinds := len(atlas.Kinds())
	return ok && l.slotAt(cursor, kinds+1) == kinds
}

// The panel with what's chosen highlighted, and the tooltip of what's under the cursor
func (p *BlueprintPlacer) DrawPanel(
	screen *ebiten.Image,
	scale float64,
	cursor utils.Vec2f,
	atlas *ecs.BlueprintAtlas,
	items *ecs.ItemAtlas,
	assets *assetmanager.AssetManager,
) error {
	screenSize := ImageSize(screen).Multiply(1 / scale)
	l, ok := p.layout(screenSize, atlas, assets)
	if !ok {
		return nil
	}
	kinds := atlas.Kinds()
	chosen := slices.Index(kinds, p.kind)
	if p.storage {
		chosen = len(kinds)
	}
	err := drawCornerPanel(screen, scale, l, len(kinds)+1, chosen, assets, func(i int, pos utils.Vec2f) error {
		if i == len(kinds) {
			drawButtonSquare(screen, scale, pos, l.slotSize, data.StorageButtonColor)
			return nil
		}
		def, _ := atlas.Def(kinds[i])
		icon, err := assets.GetAsset(def.ImageAsset())
		if err != nil {
			return fmt.Errorf("error getting the image of blueprint %d: %w", kinds[i], err)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pos.X+(l.slotSize.X-icon.FrameSize.X)/2, pos.Y+(l.slotSize.Y-icon.FrameSize.Y)/2)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(icon.Frames[0], op)
		return nil
	})
	if err != nil {
		return err
	}
	if kind, ok := p.BlueprintAt(cursor, screenSize, atlas, assets); ok {
		def, _ := atlas.Def(kind)
		return DrawTooltip(screen, scale, cursor, BlueprintTooltip(def, items), assets)
	}
	if p.StorageAt(cursor, screenSize, atlas, assets) {
		return DrawTooltip(screen, scale, cursor, []string{"Storage"}, assets)
	}
	return nil
}

func (p *BlueprintPlacer) Press(tile utils.Vec2i) {
	if p.IsActive() {
		p.dragging, p.start = true, tile
	}
}

func (p *BlueprintPlacer) IsDragging() bool {
	return p.dragging
}

// From where the drag started to tile, whichever way it went
func (p *BlueprintPlacer) Rect(tile utils.Vec2i) (first, last utils.Vec2i) {
	return utils.Vec2i{X: min(p.start.X, tile.X), Y: min(p.start.Y, tile.Y)},
		utils.Vec2i{X: max(p.start.X, tile.X), Y: max(p.start.Y, tile.Y)}
}

// Ends the drag at tile, marking the rectangle out as storage, or taking away all storage it touches
func (p *BlueprintPlacer) Release(tile utils.Vec2i, erase bool) ecs.StorageMarking {
	if !p.dragging || !p.storage {
		return ecs.StorageMarking{}
	}
	p.dragging = false
	first, last := p.Rect(tile)
	return ecs.StorageMarking{Mark: true, Erase: erase, First: first, Last: last}
}

// Ends the drag at tile, with the chosen blueprint on each of the tiles it goes on that pass the filter
func (p *BlueprintPlacer) ReleaseBlueprints(tile utils.Vec2i, fits func(utils.Vec2i) bool) ecs.BlueprintPlacement {
	if !p.dragging || p.storage {
		return ecs.BlueprintPlacement{}
	}
	tiles := slices.DeleteFunc(p.BlueprintTiles(tile), func(t utils.Vec2i) bool { return !fits(t) })
	p.dragging = false
	if len(tiles) == 0 {
		return ecs.BlueprintPlacement{}
	}
	return ecs.BlueprintPlacement{Kind: p.kind, Tiles: tiles}
}

// Where blueprints go for a drag to tile, or for the tile itself without one: the outline of the rectangle, which is a
// line when it's a tile across, row by row
func (p *BlueprintPlacer) BlueprintTiles(tile utils.Vec2i) []utils.Vec2i {
	if !p.dragging {
		return []utils.Vec2i{tile}
	}
	first, last := p.Rect(tile)
	tiles := []utils.Vec2i{}
	for y := first.Y; y <= last.Y; y++ {
		for x := first.X; x <= last.X; x++ {
			if y == first.Y || y == last.Y || x == first.X || x == last.X {
				tiles = append(tiles, utils.Vec2i{X: x, Y: y})
			}
		}
	}
	return tiles
}

// The storage being dragged out to tile: purple if it can go there, and red if it can't or it's erasing
func (p *BlueprintPlacer) DrawStorage(dst *ebiten.Image, camera utils.Vec2f, tile utils.Vec2i, valid bool) {
	if !p.dragging {
		return
	}
	c := data.StorageMarkColor
	if !valid {
		c = data.AreaUnmarkColor
	}
	first, last := p.Rect(tile)
	fillTiles(dst, camera, first, last, c)
}

// The finished tile as a blueprint on each tile it would go on for the cursor's tile: blue where it fits and red where
// it doesn't
func (p *BlueprintPlacer) Draw(
	dst *ebiten.Image,
	camera utils.Vec2f,
	tile utils.Vec2i,
	fits func(utils.Vec2i) bool,
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
	for _, t := range p.BlueprintTiles(tile) {
		pos := utils.TilePosToWorldPos(t)
		op := &colorm.DrawImageOptions{}
		op.GeoM.Translate(pos.X-camera.X-float64(img.Bounds().Dx())/2, pos.Y-camera.Y-float64(img.Bounds().Dy())/2)
		colorm.DrawImage(dst, img, blueprintColorM(fits(t)), op)
	}
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

// Its name, how much of each cost has been delivered, and how far building it has come, from 0% when it's placed, and
// whether colonists leave it alone, being off home
func SiteTooltip(site ecs.ConstructionSite, def ecs.BlueprintDef, items *ecs.ItemAtlas, home bool) []string {
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
	lines = append(lines, fmt.Sprintf("Construction %d%%", progress))
	if !home {
		lines = append(lines, "Outside the home area")
	}
	return lines
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
