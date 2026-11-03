package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"image"
	"image/color"
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

// Only the vision's square of tiles and its circle are drawn each frame, both the same size at any zoom; everything
// outside them is hidden by filling the mask first
type VisionOverlay struct {
	// One pixel per tile in the vision's square, scaled up, so that hiding every unseen tile is a single draw
	tiles      *ebiten.Image
	tilePixels []byte

	// Opaque everywhere but a hole in the middle, big enough to cover the vision's square from anywhere in the viewer's
	// tile
	circle      *ebiten.Image
	circleRadii [2]float64
	circleSize  int

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
		if o.mask != nil {
			o.mask.Deallocate()
			o.memory.Deallocate()
		}
		// Unmanaged like the world image they're drawn onto, so that they don't grow pages of the shared atlas
		unmanaged := &ebiten.NewImageOptions{Unmanaged: true}
		o.mask = ebiten.NewImageWithOptions(image.Rect(0, 0, size.X, size.Y), unmanaged)
		o.memory = ebiten.NewImageWithOptions(image.Rect(0, 0, size.X, size.Y), unmanaged)
	}

	o.mask.Fill(color.White)
	o.drawTiles(o.mask, camera, vision)
	o.drawCircle(o.mask, camera, center, vision)

	drawMemory(o.memory, camera)
	o.mask.DrawImage(o.memory, &ebiten.DrawImageOptions{Blend: ebiten.BlendSourceIn})

	screen.DrawImage(o.mask, nil)
}

// Copied over the filled mask, so that the tiles in the square replace it, clear where they're seen
func (o *VisionOverlay) drawTiles(dst *ebiten.Image, camera utils.Vec2f, vision *ecs.Vision) {
	origin, radius := vision.VisibleArea()
	side := 2*radius + 1

	if o.tiles == nil || o.tiles.Bounds().Dx() != side {
		if o.tiles != nil {
			o.tiles.Deallocate()
		}
		o.tiles = ebiten.NewImage(side, side)
		o.tilePixels = make([]byte, 4*side*side)
	}

	for y := range side {
		for x := range side {
			alpha := byte(255)
			if vision.CanSee(utils.Vec2i{X: origin.X - radius + x, Y: origin.Y - radius + y}) {
				alpha = 0
			}
			o.tilePixels[4*(y*side+x)+3] = alpha
		}
	}
	o.tiles.WritePixels(o.tilePixels)

	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(utils.Vec2i{X: origin.X - radius, Y: origin.Y - radius}).
		Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	dst.DrawImage(o.tiles, op)
}

func (o *VisionOverlay) drawCircle(dst *ebiten.Image, camera utils.Vec2f, center utils.Vec2f, vision *ecs.Vision) {
	_, radius := vision.VisibleArea()
	size := 2 * (radius + 1) * data.TileSize
	clearRadius, outer := vision.CircleRadii()

	if o.circle == nil || o.circleRadii != [2]float64{clearRadius, outer} || o.circleSize != size {
		if o.circle != nil {
			o.circle.Deallocate()
		}
		o.circle = circleImage(size, size, clearRadius, outer)
		o.circleRadii, o.circleSize = [2]float64{clearRadius, outer}, size
	}

	op := &ebiten.DrawImageOptions{Blend: blendMax}
	op.GeoM.Translate(center.X-camera.X-float64(size)/2, center.Y-camera.Y-float64(size)/2)
	dst.DrawImage(o.circle, op)
}

// Whether the overlay leaves the world position at least partly uncovered
func Revealed(pos, center utils.Vec2f, vision *ecs.Vision) bool {
	_, outer := vision.CircleRadii()
	return vision.CanSee(utils.WorldPosToTilePos(pos)) && pos.Subtract(center).Length() < outer
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
