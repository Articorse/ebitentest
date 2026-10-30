package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

func DrawChunks(
	screen *ebiten.Image,
	camera utils.Vec2f,
	chunkCont *ChunkContainer,
) error {
	marginBakesLeft := data.MaxChunkImageBakesPerFrame

	for ck, c := range chunkCont.chunks {
		cBounds := chunkDrawBounds(ck)

		if !utils.DetectAABBCollision(cBounds, cameraBounds(camera, data.ChunkImageFreeMarginPx)) {
			c.freeImage()
			continue
		}

		visible := utils.DetectAABBCollision(cBounds, cameraBounds(camera, 0))

		if c.Image == nil {
			inBakeMargin := utils.DetectAABBCollision(cBounds, cameraBounds(camera, data.ChunkImageBakeMarginPx))
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

func cameraBounds(camera utils.Vec2f, margin float64) [2]utils.Vec2f {
	return [2]utils.Vec2f{
		{X: camera.X - margin, Y: camera.Y - margin},
		{X: camera.X + data.CameraWidth + margin, Y: camera.Y + data.CameraHeight + margin},
	}
}

func (x *chunk) generateChunkImage(atlas map[data.TileEnum]tileDef) error {
	img := ebiten.NewImage(data.ChunkSize*data.TileSize, data.ChunkSize*data.TileSize)

	for tCell, tEnum := range x.tiles {
		tile, ok := atlas[tEnum]
		if !ok {
			img.Deallocate()
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}

		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(tCell%data.ChunkSize*data.TileSize), float64(tCell/data.ChunkSize*data.TileSize))
		img.DrawImage(tile.image, &opts)
	}

	x.Image = img

	return nil
}

func (x *chunk) freeImage() {
	if x.Image == nil {
		return
	}

	x.Image.Deallocate()
	x.Image = nil
}
