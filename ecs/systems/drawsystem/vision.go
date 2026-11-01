package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

// One pixel per on-screen tile, scaled up, so that hiding every unseen tile is a single draw
type VisionOverlay struct {
	img    *ebiten.Image
	pixels []byte
}

func (o *VisionOverlay) Draw(screen *ebiten.Image, camera utils.Vec2f, vision *ecs.Vision) {
	size := screen.Bounds().Size()
	minTile := utils.WorldPosToTilePos(camera)
	maxTile := utils.WorldPosToTilePos(camera.Add(utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}))
	w, h := maxTile.X-minTile.X+1, maxTile.Y-minTile.Y+1

	if o.img == nil || o.img.Bounds().Dx() != w || o.img.Bounds().Dy() != h {
		o.img = ebiten.NewImage(w, h)
		o.pixels = make([]byte, 4*w*h)
	}

	for y := range h {
		for x := range w {
			alpha := byte(255)
			if vision.CanSee(utils.Vec2i{X: minTile.X + x, Y: minTile.Y + y}) {
				alpha = 0
			}
			// Premultiplied, so black only needs its alpha set
			o.pixels[4*(y*w+x)+3] = alpha
		}
	}
	o.img.WritePixels(o.pixels)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(data.TileSize, data.TileSize)
	topLeft := utils.TilePosToWorldPos(minTile).Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
	screen.DrawImage(o.img, op)
}
