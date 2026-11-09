package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type areaLook struct {
	name                string
	shown, mark, button color.RGBA
}

var areaLooks = [data.AreaCount]areaLook{
	data.Area_Home:    {name: "Home", shown: data.HomeAreaColor, mark: data.HomeMarkColor, button: data.HomeButtonColor},
	data.Area_Harvest: {name: "Harvest", shown: data.HarvestAreaColor, mark: data.HarvestMarkColor, button: data.HarvestButtonColor},
	data.Area_Storage: {name: "Storage", shown: data.StorageAreaColor, mark: data.StorageMarkColor, button: data.StorageButtonColor},
	data.Area_Roof:    {name: "Roof", shown: data.RoofAreaColor, mark: data.RoofMarkColor, button: data.RoofButtonColor},
	data.Area_NoRoof:  {name: "No roof", shown: data.NoRoofAreaColor, mark: data.NoRoofMarkColor, button: data.NoRoofButtonColor},
}

// In the order of the area menu's buttons, which storage, in build mode, isn't one of
func Areas() []data.AreaEnum {
	return []data.AreaEnum{data.Area_Home, data.Area_Harvest, data.Area_Roof, data.Area_NoRoof}
}

// The area menu: the area chosen, marked out by dragging rectangles over tiles, and the panel of areas to choose from
// in the bottom right corner while it's open. Like the placer, it only changes what's drawn and what's sent.
type AreaPainter struct {
	area     data.AreaEnum
	dragging bool
	start    utils.Vec2i
}

func (p *AreaPainter) Area() (data.AreaEnum, bool) {
	return p.area, p.area != data.Area_None
}

func (p *AreaPainter) IsActive() bool {
	return p.area != data.Area_None
}

// Opens with the first area chosen
func (p *AreaPainter) Toggle() {
	if p.IsActive() {
		p.Stop()
		return
	}
	p.area = Areas()[0]
}

func (p *AreaPainter) Choose(area data.AreaEnum) {
	p.area, p.dragging = area, false
}

func (p *AreaPainter) Stop() {
	p.area, p.dragging = data.Area_None, false
}

func (p *AreaPainter) layout(screenSize utils.Vec2f, assets *assetmanager.AssetManager) (cornerLayout, bool) {
	slot, err := assets.GetAsset(common.AssetImageSlot)
	if !p.IsActive() || err != nil {
		return cornerLayout{}, false
	}
	return newCornerLayout(screenSize, len(Areas()), slot.FrameSize), true
}

// Whether clicks at the cursor are the panel's rather than the world's
func (p *AreaPainter) Covers(cursor, screenSize utils.Vec2f, assets *assetmanager.AssetManager) bool {
	l, ok := p.layout(screenSize, assets)
	return ok && l.covers(cursor)
}

// The area under the cursor on the panel
func (p *AreaPainter) AreaAt(cursor, screenSize utils.Vec2f, assets *assetmanager.AssetManager) (data.AreaEnum, bool) {
	l, ok := p.layout(screenSize, assets)
	if !ok {
		return data.Area_None, false
	}
	areas := Areas()
	if i := l.slotAt(cursor, len(areas)); i != -1 {
		return areas[i], true
	}
	return data.Area_None, false
}

// The panel with each area's colour on its button and the chosen one highlighted, and the name of the area under the
// cursor
func (p *AreaPainter) DrawPanel(screen *ebiten.Image, scale float64, cursor utils.Vec2f, assets *assetmanager.AssetManager) error {
	screenSize := ImageSize(screen).Multiply(1 / scale)
	l, ok := p.layout(screenSize, assets)
	if !ok {
		return nil
	}
	areas := Areas()
	chosen := -1
	for i, a := range areas {
		if a == p.area {
			chosen = i
		}
	}
	err := drawCornerPanel(screen, scale, l, len(areas), chosen, assets, func(i int, pos utils.Vec2f) error {
		drawButtonSquare(screen, scale, pos, l.slotSize, areaLooks[areas[i]].button)
		return nil
	})
	if err != nil {
		return err
	}
	if a, ok := p.AreaAt(cursor, screenSize, assets); ok {
		return DrawTooltip(screen, scale, cursor, []string{areaLooks[a].name}, assets)
	}
	return nil
}

func (p *AreaPainter) Press(tile utils.Vec2i) {
	if p.IsActive() {
		p.dragging, p.start = true, tile
	}
}

func (p *AreaPainter) IsDragging() bool {
	return p.dragging
}

// From where the drag started to tile, whichever way it went
func (p *AreaPainter) rect(tile utils.Vec2i) (first, last utils.Vec2i) {
	return utils.Vec2i{X: min(p.start.X, tile.X), Y: min(p.start.Y, tile.Y)},
		utils.Vec2i{X: max(p.start.X, tile.X), Y: max(p.start.Y, tile.Y)}
}

// Ends the drag at tile, putting the rectangle in the chosen area, or taking it out
func (p *AreaPainter) Release(tile utils.Vec2i, erase bool) ecs.AreaMarking {
	if !p.dragging {
		return ecs.AreaMarking{}
	}
	p.dragging = false
	first, last := p.rect(tile)
	return ecs.AreaMarking{Area: p.area, Erase: erase, First: first, Last: last}
}

// The rectangle being dragged out to tile, in the area's colour to mark or red to erase
func (p *AreaPainter) Draw(dst *ebiten.Image, camera utils.Vec2f, tile utils.Vec2i, erase bool) {
	if !p.dragging {
		return
	}
	c := areaLooks[p.area].mark
	if erase {
		c = data.AreaUnmarkColor
	}
	first, last := p.rect(tile)
	fillTiles(dst, camera, first, last, c)
}

// A button's colour, in from the edges of its slot
func drawButtonSquare(screen *ebiten.Image, scale float64, slotPos, slotSize utils.Vec2f, c color.Color) {
	inset := float64(data.AreaButtonInsetPx)
	x, y := (slotPos.X+inset)*scale, (slotPos.Y+inset)*scale
	w, h := (slotSize.X-2*inset)*scale, (slotSize.Y-2*inset)*scale
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), c, false)
}

// The area's tiles in view, since areas are only shown while they're being marked out or built on
func DrawArea(dst *ebiten.Image, camera utils.Vec2f, area data.AreaEnum, inArea func(data.AreaEnum, utils.Vec2i) bool) {
	drawTilesWhere(dst, camera, func(t utils.Vec2i) bool { return inArea(area, t) }, areaLooks[area].shown)
}

func drawTilesWhere(dst *ebiten.Image, camera utils.Vec2f, in func(utils.Vec2i) bool, c color.Color) {
	first, last := utils.WorldPosToTilePos(camera), utils.WorldPosToTilePos(camera.Add(ImageSize(dst)))
	for y := first.Y; y <= last.Y; y++ {
		// Runs along the row are drawn as one rectangle, which is far fewer to draw
		for x := first.X; x <= last.X; x++ {
			if !in(utils.Vec2i{X: x, Y: y}) {
				continue
			}
			end := x
			for end+1 <= last.X && in(utils.Vec2i{X: end + 1, Y: y}) {
				end++
			}
			fillTiles(dst, camera, utils.Vec2i{X: x, Y: y}, utils.Vec2i{X: end, Y: y}, c)
			x = end
		}
	}
}

// A tile's position is its center, so the rectangle starts half a tile before the first one's
func fillTiles(dst *ebiten.Image, camera utils.Vec2f, first, last utils.Vec2i, c color.Color) {
	topLeft := utils.TilePosToWorldPos(first).Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2}).Subtract(camera)
	x, y := topLeft.X, topLeft.Y
	w, h := float64((last.X-first.X+1)*data.TileSize), float64((last.Y-first.Y+1)*data.TileSize)
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

// Over every tile with air, from red with little oxygen per tile, through yellow, to green with plenty
func DrawOxygen(dst *ebiten.Image, camera utils.Vec2f, oxygenAt func(utils.Vec2i) (int64, bool)) {
	first, last := utils.WorldPosToTilePos(camera), utils.WorldPosToTilePos(camera.Add(ImageSize(dst)))
	shade := func(t utils.Vec2i) (color.RGBA, bool) {
		mg, ok := oxygenAt(t)
		if !ok {
			return color.RGBA{}, false
		}
		return OxygenColor(mg), true
	}
	for y := first.Y; y <= last.Y; y++ {
		// Runs of a colour along the row are drawn as one rectangle, which is far fewer to draw
		for x := first.X; x <= last.X; x++ {
			c, ok := shade(utils.Vec2i{X: x, Y: y})
			if !ok {
				continue
			}
			end := x
			for end+1 <= last.X {
				next, ok := shade(utils.Vec2i{X: end + 1, Y: y})
				if !ok || next != c {
					break
				}
				end++
			}
			fillTiles(dst, camera, utils.Vec2i{X: x, Y: y}, utils.Vec2i{X: end, Y: y}, c)
			x = end
		}
	}
}

// Premultiplied, in steps, so that tiles alike draw as one
func OxygenColor(mg int64) color.RGBA {
	const steps = 8
	f := float64(mg-data.OverlayOxygenRedMg) / float64(data.OverlayOxygenGreenMg-data.OverlayOxygenRedMg)
	f = math.Round(max(0, min(1, f))*steps) / steps
	r, g := 1.0, 1.0
	if f < 0.5 {
		g = 2 * f
	} else {
		r = 2 * (1 - f)
	}
	a := float64(data.OverlayOxygenAlpha)
	return color.RGBA{R: uint8(r * a), G: uint8(g * a), A: uint8(a)}
}
