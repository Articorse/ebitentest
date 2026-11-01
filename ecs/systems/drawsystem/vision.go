package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type VisionOverlay struct {
	// One pixel per on-screen tile, scaled up, so that hiding every unseen tile is a single draw
	tiles      *ebiten.Image
	tilePixels []byte

	// Dark everywhere but a hole in the middle, twice the screen's size so that it covers the screen wherever the
	// hole is placed
	circle       *ebiten.Image
	circleRadius int
	circleSize   [2]int
}

// Draws the unseen tiles dark, and everything beyond a smooth circle around center. Tile edges only move when the
// viewer crosses into another tile, which the lagging camera turns into jitter; the circle moves with the viewer.
func (o *VisionOverlay) Draw(screen *ebiten.Image, camera utils.Vec2f, center utils.Vec2f, vision *ecs.Vision) {
	o.drawTiles(screen, camera, vision)
	o.drawCircle(screen, camera, center, vision.Radius())
}

func (o *VisionOverlay) drawTiles(screen *ebiten.Image, camera utils.Vec2f, vision *ecs.Vision) {
	size := screen.Bounds().Size()
	minTile := utils.WorldPosToTilePos(camera)
	maxTile := utils.WorldPosToTilePos(camera.Add(utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}))
	w, h := maxTile.X-minTile.X+1, maxTile.Y-minTile.Y+1

	if o.tiles == nil || o.tiles.Bounds().Dx() != w || o.tiles.Bounds().Dy() != h {
		o.tiles = ebiten.NewImage(w, h)
		o.tilePixels = make([]byte, 4*w*h)
	}

	for y := range h {
		for x := range w {
			alpha := byte(255)
			if vision.CanSee(utils.Vec2i{X: minTile.X + x, Y: minTile.Y + y}) {
				alpha = 0
			}
			// Premultiplied, so black only needs its alpha set
			o.tilePixels[4*(y*w+x)+3] = alpha
		}
	}
	o.tiles.WritePixels(o.tilePixels)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(minTile).Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	screen.DrawImage(o.tiles, op)
}

func (o *VisionOverlay) drawCircle(screen *ebiten.Image, camera utils.Vec2f, center utils.Vec2f, radius int) {
	size := screen.Bounds().Size()
	w, h := 2*size.X, 2*size.Y

	if o.circle == nil || o.circleRadius != radius || o.circleSize != [2]int{w, h} {
		o.circle = circleImage(w, h, radius)
		o.circleRadius, o.circleSize = radius, [2]int{w, h}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(center.X-camera.X-float64(w)/2, center.Y-camera.Y-float64(h)/2)
	screen.DrawImage(o.circle, op)
}

func circleImage(w, h, radius int) *ebiten.Image {
	// Shrunk so that every tile the circle shows was computed: the circle is centered on the viewer, who can stand
	// up to half a tile diagonal away from the center of the tile their vision is computed from
	outer := max(float64(radius)-data.VisionCircleMarginTiles, data.VisionCircleFadeTiles) * data.TileSize
	inner := outer - data.VisionCircleFadeTiles*data.TileSize

	pixels := make([]byte, 4*w*h)
	cx, cy := float64(w)/2, float64(h)/2
	for y := range h {
		for x := range w {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			t := min(max((d-inner)/(outer-inner), 0), 1)
			pixels[4*(y*w+x)+3] = byte(math.Round(t * 255))
		}
	}

	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}
