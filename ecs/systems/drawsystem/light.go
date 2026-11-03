package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var blendMultiply = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

// How bright each light level is drawn: in proportion to it, but never below data.MinLightBrightness, so that what's
// seen in dim light doesn't look much darker than what's remembered next to it. Unlit tiles are covered by memory.
var lightBrightness = func() (b [data.MaxLight + 1]byte) {
	for light := range b {
		b[light] = byte(math.Round(max(float64(light)/data.MaxLight, data.MinLightBrightness) * 255))
	}
	return b
}()

// Dims everything drawn by the light on it, so that what's seen grows darker gradually, such as into a cave
type LightOverlay struct {
	// One pixel per tile of the light map, scaled up and filtered between tile centers, so that the light changes
	// smoothly rather than at tile edges
	image  *ebiten.Image
	pixels []byte
}

// Only covers the light map's square, since what's beyond it can't be seen and is covered by memory
func (o *LightOverlay) Draw(world *ebiten.Image, camera utils.Vec2f, light *ecs.LightMap) {
	center, radius := light.Area()
	side := 2*radius + 1

	if o.image == nil || o.image.Bounds().Dx() != side {
		if o.image != nil {
			o.image.Deallocate()
		}
		o.image = ebiten.NewImage(side, side)
		o.pixels = make([]byte, 4*side*side)
	}

	for y := range side {
		for x := range side {
			b := lightBrightness[light.At(utils.Vec2i{X: center.X - radius + x, Y: center.Y - radius + y})]
			i := 4 * (y*side + x)
			o.pixels[i], o.pixels[i+1], o.pixels[i+2], o.pixels[i+3] = b, b, b, 255
		}
	}
	o.image.WritePixels(o.pixels)

	op := &ebiten.DrawImageOptions{Blend: blendMultiply, Filter: ebiten.FilterLinear}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(utils.Vec2i{X: center.X - radius, Y: center.Y - radius}).
		Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	world.DrawImage(o.image, op)
}
