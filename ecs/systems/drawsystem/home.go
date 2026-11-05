package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Marking out home by dragging rectangles over tiles. Like the placer, it only changes what's drawn and what's sent.
type HomePainter struct {
	active   bool
	dragging bool
	start    utils.Vec2i
}

func (p *HomePainter) IsActive() bool {
	return p.active
}

func (p *HomePainter) Toggle() {
	p.active, p.dragging = !p.active, false
}

func (p *HomePainter) Stop() {
	p.active, p.dragging = false, false
}

func (p *HomePainter) Press(tile utils.Vec2i) {
	if p.active {
		p.dragging, p.start = true, tile
	}
}

func (p *HomePainter) IsDragging() bool {
	return p.dragging
}

// From where the drag started to tile, whichever way it went
func (p *HomePainter) rect(tile utils.Vec2i) (first, last utils.Vec2i) {
	return utils.Vec2i{X: min(p.start.X, tile.X), Y: min(p.start.Y, tile.Y)},
		utils.Vec2i{X: max(p.start.X, tile.X), Y: max(p.start.Y, tile.Y)}
}

// Ends the drag at tile, marking the rectangle as home, or unmarking it
func (p *HomePainter) Release(tile utils.Vec2i, unmark bool) ecs.HomeMarking {
	if !p.dragging {
		return ecs.HomeMarking{}
	}
	p.dragging = false
	first, last := p.rect(tile)
	mark := ecs.Home_Mark
	if unmark {
		mark = ecs.Home_Unmark
	}
	return ecs.HomeMarking{Mark: mark, First: first, Last: last}
}

// The rectangle being dragged out to tile, green to mark or red to unmark
func (p *HomePainter) Draw(dst *ebiten.Image, camera utils.Vec2f, tile utils.Vec2i, unmark bool) {
	if !p.dragging {
		return
	}
	c := data.HomeMarkColor
	if unmark {
		c = data.HomeUnmarkColor
	}
	first, last := p.rect(tile)
	fillTiles(dst, camera, first, last, c)
}

// The home tiles in view, since home is only shown while it's being marked out or built on
func DrawHomeArea(dst *ebiten.Image, camera utils.Vec2f, isHome func(utils.Vec2i) bool) {
	first, last := utils.WorldPosToTilePos(camera), utils.WorldPosToTilePos(camera.Add(ImageSize(dst)))
	for y := first.Y; y <= last.Y; y++ {
		// Runs of home along the row are drawn as one rectangle, which is far fewer to draw
		for x := first.X; x <= last.X; x++ {
			if !isHome(utils.Vec2i{X: x, Y: y}) {
				continue
			}
			end := x
			for end+1 <= last.X && isHome(utils.Vec2i{X: end + 1, Y: y}) {
				end++
			}
			fillTiles(dst, camera, utils.Vec2i{X: x, Y: y}, utils.Vec2i{X: end, Y: y}, data.HomeAreaColor)
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
