package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawChunks(
	screen *ebiten.Image,
	camera utils.Vec2f,
	chunkCont *ChunkContainer,
) error {
	marginBakesLeft := data.MaxChunkImageBakesPerFrame
	size := screen.Bounds().Size()
	view := utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}

	for ck, c := range chunkCont.chunks {
		cBounds := chunkDrawBounds(ck)

		if !utils.DetectAABBCollision(cBounds, cameraBounds(camera, view, data.ChunkImageFreeMarginPx)) {
			c.freeImage()
			continue
		}

		visible := utils.DetectAABBCollision(cBounds, cameraBounds(camera, view, 0))

		if c.Image == nil {
			inBakeMargin := utils.DetectAABBCollision(cBounds, cameraBounds(camera, view, data.ChunkImageBakeMarginPx))
			if !visible && (!inBakeMargin || marginBakesLeft <= 0) {
				continue
			}

			if err := c.generateChunkImage(chunkCont.atlas); err != nil {
				return fmt.Errorf("failed to generate image of chunk %v: %w", ck, err)
			}

			if !visible {
				marginBakesLeft--
			}
		}

		if !visible {
			continue
		}

		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Translate(cBounds[0].X-camera.X, cBounds[0].Y-camera.Y)
		screen.DrawImage(c.Image, &opts)
	}

	return nil
}

func chunkDrawBounds(ck utils.Vec2i) [2]utils.Vec2f {
	minX := float64(ck.X)*data.ChunkSize*data.TileSize - data.TileSize/2
	minY := float64(ck.Y)*data.ChunkSize*data.TileSize - data.TileSize/2

	return [2]utils.Vec2f{
		{X: minX, Y: minY},
		{X: minX + data.ChunkSize*data.TileSize, Y: minY + data.ChunkSize*data.TileSize},
	}
}

func cameraBounds(camera, view utils.Vec2f, margin float64) [2]utils.Vec2f {
	return [2]utils.Vec2f{
		{X: camera.X - margin, Y: camera.Y - margin},
		{X: camera.X + view.X + margin, Y: camera.Y + view.Y + margin},
	}
}

func (x *chunk) generateChunkImage(atlas map[data.TileEnum]tileDef) error {
	// Unmanaged, since a big image on Ebitengine's shared atlas grows its page, up to the GPU's largest texture size,
	// which the GPU can fail to allocate
	side := data.ChunkSize * data.TileSize
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, side, side), &ebiten.NewImageOptions{Unmanaged: true})

	for tCell, tEnum := range x.tiles {
		tile, ok := atlas[tEnum]
		if !ok {
			img.Deallocate()
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}

		drawTileInto(img, tCell, tile.image)
	}

	x.Image = img

	return nil
}

// Replaces what's drawn on the tile, so that drawing a changed tile over the old one doesn't blend with it
func drawTileInto(chunkImage *ebiten.Image, idx int, tile *ebiten.Image) {
	opts := ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
	opts.GeoM.Translate(float64(idx%data.ChunkSize*data.TileSize), float64(idx/data.ChunkSize*data.TileSize))
	chunkImage.DrawImage(tile, &opts)
}

func (x *chunk) freeImage() {
	if x.Image == nil {
		return
	}

	x.Image.Deallocate()
	x.Image = nil
}

// Outlines the colliders on screen of the features on the tiles that are included
func (cc *ChunkContainer) DrawFeatureColliders(
	screen *ebiten.Image,
	camera utils.Vec2f,
	lineColor color.RGBA,
	include func(utils.Vec2i) bool,
) {
	size := screen.Bounds().Size()
	onScreen := cameraBounds(camera, utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}, 0)
	minTile, maxTile := utils.WorldPosToTilePos(onScreen[0]), utils.WorldPosToTilePos(onScreen[1])

	for ck, c := range cc.chunks {
		chunkMin := utils.Vec2i{X: ck.X * data.ChunkSize, Y: ck.Y * data.ChunkSize}
		chunkMax := utils.Vec2i{X: chunkMin.X + data.ChunkSize - 1, Y: chunkMin.Y + data.ChunkSize - 1}
		if chunkMax.X < minTile.X || chunkMin.X > maxTile.X || chunkMax.Y < minTile.Y || chunkMin.Y > maxTile.Y {
			continue
		}

		for t := range c.features {
			bounds, ok := cc.featureCollider(c, t)
			if !ok || !include(t) || !utils.DetectAABBCollision(bounds, onScreen) {
				continue
			}

			size := bounds[1].Subtract(bounds[0])
			vector.StrokeRect(screen, float32(bounds[0].X-camera.X), float32(bounds[0].Y-camera.Y), float32(size.X), float32(size.Y), 1, lineColor, false)
		}
	}
}

// Outlines the chunks on screen that are active (with or without a baked image), loaded but inactive, or being loaded.
// Inset by a pixel, so that the borders of neighbors are side by side rather than drawn over each other.
func (cc *ChunkContainer) DrawChunkBorders(screen *ebiten.Image, camera utils.Vec2f) {
	size := screen.Bounds().Size()
	onScreen := cameraBounds(camera, utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}, 0)
	minChunk, maxChunk := utils.WorldPosToChunkGridPos(onScreen[0]), utils.WorldPosToChunkGridPos(onScreen[1])

	for y := minChunk.Y - 1; y <= maxChunk.Y+1; y++ {
		for x := minChunk.X - 1; x <= maxChunk.X+1; x++ {
			pos := utils.Vec2i{X: x, Y: y}
			lineColor, ok := cc.chunkStateColor(pos)
			bounds := chunkDrawBounds(pos)
			if !ok || !utils.DetectAABBCollision(bounds, onScreen) {
				continue
			}

			side := float32(bounds[1].X - bounds[0].X - 2)
			vector.StrokeRect(screen, float32(bounds[0].X-camera.X+1), float32(bounds[0].Y-camera.Y+1), side, side, 1, lineColor, false)
		}
	}
}

func (cc *ChunkContainer) chunkStateColor(pos utils.Vec2i) (color.RGBA, bool) {
	if c, ok := cc.chunks[pos]; ok {
		if c.Image == nil {
			return data.Debug_ChunkUnbakedColor, true
		}
		return data.Debug_ChunkActiveColor, true
	}
	if cc.isAvailable(pos) {
		return data.Debug_ChunkInactiveColor, true
	}
	if _, ok := cc.loadsInFlight[pos]; ok {
		return data.Debug_ChunkLoadingColor, true
	}
	return color.RGBA{}, false
}
