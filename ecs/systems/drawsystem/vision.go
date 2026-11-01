package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var blendMax = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOne,
	BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationMax,
	BlendOperationAlpha:         ebiten.BlendOperationMax,
}

type VisionOverlay struct {
	// One pixel per on-screen tile, scaled up, so that hiding every unseen tile is a single draw
	tiles      *ebiten.Image
	tilePixels []byte

	// Opaque everywhere but a hole in the middle, twice the screen's size so that it covers the screen wherever the
	// hole is placed
	circle      *ebiten.Image
	circleRadii [2]float64
	circleSize  [2]int

	// How much of the remembered view replaces the live one, per pixel
	mask   *ebiten.Image
	memory *ebiten.Image
}

// Replaces the live view with the remembered one wherever the viewer can't see, or is outside their smooth vision
// circle. Tile edges only move when the viewer crosses into another tile, which the lagging camera turns into jitter;
// the circle moves with the viewer.
func (o *VisionOverlay) Draw(
	screen *ebiten.Image,
	camera utils.Vec2f,
	center utils.Vec2f,
	vision *ecs.Vision,
	drawMemory func(dst *ebiten.Image, camera utils.Vec2f),
) {
	size := screen.Bounds().Size()
	if o.mask == nil || o.mask.Bounds().Size() != size {
		o.mask = ebiten.NewImage(size.X, size.Y)
		o.memory = ebiten.NewImage(size.X, size.Y)
	}

	o.mask.Clear()
	o.drawTiles(o.mask, camera, vision)
	o.drawCircle(o.mask, camera, center, vision)

	drawMemory(o.memory, camera)
	o.mask.DrawImage(o.memory, &ebiten.DrawImageOptions{Blend: ebiten.BlendSourceIn})

	screen.DrawImage(o.mask, nil)
}

func (o *VisionOverlay) drawTiles(dst *ebiten.Image, camera utils.Vec2f, vision *ecs.Vision) {
	size := dst.Bounds().Size()
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
			o.tilePixels[4*(y*w+x)+3] = alpha
		}
	}
	o.tiles.WritePixels(o.tilePixels)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(minTile).Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	dst.DrawImage(o.tiles, op)
}

func (o *VisionOverlay) drawCircle(dst *ebiten.Image, camera utils.Vec2f, center utils.Vec2f, vision *ecs.Vision) {
	size := dst.Bounds().Size()
	w, h := 2*size.X, 2*size.Y
	clearRadius, outer := vision.CircleRadii()

	if o.circle == nil || o.circleRadii != [2]float64{clearRadius, outer} || o.circleSize != [2]int{w, h} {
		if o.circle != nil {
			o.circle.Deallocate()
		}
		o.circle = circleImage(w, h, clearRadius, outer)
		o.circleRadii, o.circleSize = [2]float64{clearRadius, outer}, [2]int{w, h}
	}

	op := &ebiten.DrawImageOptions{Blend: blendMax}
	op.GeoM.Translate(center.X-camera.X-float64(w)/2, center.Y-camera.Y-float64(h)/2)
	dst.DrawImage(o.circle, op)
}

func circleImage(w, h int, clearRadius, outer float64) *ebiten.Image {
	pixels := make([]byte, 4*w*h)
	cx, cy := float64(w)/2, float64(h)/2
	for y := range h {
		for x := range w {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			t := min(max((d-clearRadius)/(outer-clearRadius), 0), 1)
			pixels[4*(y*w+x)+3] = byte(math.Round(t * 255))
		}
	}

	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}
