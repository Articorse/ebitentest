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

// Build mode: the blueprint chosen, to be put down with a click, and the panel of blueprints to choose from in the
// bottom right corner while it's on. Like the inventory menu, it only changes what's drawn and what a click sends.
// Positions on the panel are in UI pixels.
type BlueprintPlacer struct {
	kind data.BlueprintEnum
}

func (p *BlueprintPlacer) Start(kind data.BlueprintEnum) {
	p.kind = kind
}

// Starts with the first blueprint chosen
func (p *BlueprintPlacer) Toggle(atlas *ecs.BlueprintAtlas) {
	if _, ok := p.Kind(); ok {
		p.Stop()
		return
	}
	if kinds := atlas.Kinds(); len(kinds) > 0 {
		p.Start(kinds[0])
	}
}

func (p *BlueprintPlacer) Stop() {
	p.kind = data.Blueprint_None
}

func (p *BlueprintPlacer) Kind() (data.BlueprintEnum, bool) {
	return p.kind, p.kind != data.Blueprint_None
}

func (p *BlueprintPlacer) layout(screenSize utils.Vec2f, atlas *ecs.BlueprintAtlas, assets *assetmanager.AssetManager) (cornerLayout, bool) {
	slot, err := assets.GetAsset(common.AssetImageSlot)
	if _, on := p.Kind(); !on || err != nil {
		return cornerLayout{}, false
	}
	return newCornerLayout(screenSize, len(atlas.Kinds()), slot.FrameSize), true
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

// The panel with the chosen blueprint's slot highlighted, and the tooltip of the blueprint under the cursor
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
	if err := drawBlueprintPanel(screen, scale, l, atlas, p.kind, assets); err != nil {
		return err
	}
	if kind, ok := p.BlueprintAt(cursor, screenSize, atlas, assets); ok {
		def, _ := atlas.Def(kind)
		return DrawTooltip(screen, scale, cursor, BlueprintTooltip(def, items), assets)
	}
	return nil
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
func drawBlueprintPanel(screen *ebiten.Image, scale float64, l cornerLayout, atlas *ecs.BlueprintAtlas, chosen data.BlueprintEnum, assets *assetmanager.AssetManager) error {
	kinds := atlas.Kinds()
	return drawCornerPanel(screen, scale, l, len(kinds), slices.Index(kinds, chosen), assets, func(i int, p utils.Vec2f) error {
		def, _ := atlas.Def(kinds[i])
		icon, err := assets.GetAsset(def.ImageAsset())
		if err != nil {
			return fmt.Errorf("error getting the image of blueprint %d: %w", kinds[i], err)
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(p.X+(l.slotSize.X-icon.FrameSize.X)/2, p.Y+(l.slotSize.Y-icon.FrameSize.Y)/2)
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(icon.Frames[0], op)
		return nil
	})
}
