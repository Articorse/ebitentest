package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"image"
	"image/color"
	"log"

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

	// Draws the circle, which changes size every tick at dusk and dawn, so it's computed per pixel on the GPU rather
	// than kept as an image that would have to be rebuilt each time
	circleShader *ebiten.Shader
	shaderFailed bool

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

// Opaque everywhere but a hole around the viewer, over an area big enough to cover the vision's square from anywhere
// in the viewer's tile
func (o *VisionOverlay) drawCircle(dst *ebiten.Image, camera utils.Vec2f, center utils.Vec2f, vision *ecs.Vision) {
	if o.circleShader == nil && !o.shaderFailed {
		shader, err := ebiten.NewShader(circleShaderSrc)
		if err != nil {
			log.Printf("error compiling the vision circle shader, so the circle isn't drawn: %v", err)
			o.shaderFailed = true
			return
		}
		o.circleShader = shader
	}
	if o.circleShader == nil {
		return
	}

	_, radius := vision.VisibleArea()
	size := 2 * (radius + 1) * data.TileSize
	clearRadius, outer := vision.CircleRadii()
	onDst := center.Subtract(camera)

	op := &ebiten.DrawRectShaderOptions{Blend: blendMax}
	op.GeoM.Translate(onDst.X-float64(size)/2, onDst.Y-float64(size)/2)
	op.Uniforms = map[string]any{
		"Center":      []float32{float32(onDst.X), float32(onDst.Y)},
		"ClearRadius": float32(clearRadius),
		"Outer":       float32(outer),
	}
	dst.DrawRectShader(size, size, o.circleShader, op)
}

// Clear within ClearRadius of Center, fading to opaque at Outer. Positions are in the destination's pixels, and its
// alpha is all that matters.
var circleShaderSrc = []byte(`//kage:unit pixels

package main

var Center vec2
var ClearRadius float
var Outer float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	t := clamp((distance(dst.xy, Center)-ClearRadius)/(Outer-ClearRadius), 0, 1)
	return vec4(t)
}
`)

// Whether the overlay leaves the world position at least partly uncovered
func Revealed(pos, center utils.Vec2f, vision *ecs.Vision) bool {
	_, outer := vision.CircleRadii()
	return vision.CanSee(utils.WorldPosToTilePos(pos)) && pos.Subtract(center).Length() < outer
}
