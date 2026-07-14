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
	for ck, c := range chunkCont.chunks {
		cPosXMin := float64(ck.X) * data.ChunkSize * data.TileSize
		cPosYMin := float64(ck.Y) * data.ChunkSize * data.TileSize
		cPosXMax := cPosXMin + data.ChunkSize*data.TileSize
		cPosYMax := cPosYMin + data.ChunkSize*data.TileSize

		if cPosXMax < camera.X || cPosXMin > camera.X+float64(data.CameraWidth) ||
			cPosYMax < camera.Y || cPosYMin > camera.Y+float64(data.CameraHeight) {
			continue
		}

		opts := ebiten.DrawImageOptions{}
		posX := cPosXMin - camera.X - data.TileSize/2
		posY := cPosYMin - camera.Y - data.TileSize/2
		opts.GeoM.Translate(posX, posY)
		screen.DrawImage(c.Image, &opts)
	}

	return nil
}

func (x *chunk) generateChunkImage(atlas map[data.TileEnum]tileDef) error {
	x.Image = ebiten.NewImage(data.ChunkSize*data.TileSize, data.ChunkSize*data.TileSize)

	for tCell, tEnum := range x.tiles {
		tile, ok := atlas[tEnum]
		if !ok {
			return fmt.Errorf("tile enum %d not found in atlas", tEnum)
		}

		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(tCell%data.ChunkSize*data.TileSize), float64(tCell/data.ChunkSize*data.TileSize))
		x.Image.DrawImage(tile.image, &opts)
	}

	return nil
}
