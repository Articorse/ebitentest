package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Only the vision's square of tiles is drawn each frame, the same size at any zoom; everything outside it is hidden by
// filling the mask first
type VisionOverlay struct {
	// One pixel per tile in the vision's square, scaled up, so that hiding every unseen tile is a single draw
	tiles      *ebiten.Image
	tilePixels []byte

	// How much of the remembered view replaces the live one, per pixel
	mask   *ebiten.Image
	memory *ebiten.Image
}

// Replaces the live view with the remembered one wherever the viewer can't see
func (o *VisionOverlay) Draw(
	screen *ebiten.Image,
	camera utils.Vec2f,
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

	drawMemory(o.memory, camera)
	o.mask.DrawImage(o.memory, &ebiten.DrawImageOptions{Blend: ebiten.BlendSourceIn})

	screen.DrawImage(o.mask, nil)
}

// Copied over the filled mask, so that the tiles in the square replace it, clear where they're seen. Filtered between
// tile centers, so that the edge of what's seen is soft rather than following tile edges.
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

	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy, Filter: ebiten.FilterLinear}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(utils.Vec2i{X: origin.X - radius, Y: origin.Y - radius}).
		Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	dst.DrawImage(o.tiles, op)
}

// Whether the overlay leaves the world position uncovered
func Revealed(pos utils.Vec2f, vision *ecs.Vision) bool {
	return vision.CanSee(utils.WorldPosToTilePos(pos))
}
