package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
)

func DrawChunks(
	screen *ebiten.Image,
	camera utils.Vec2,
	chunkCont *ChunkContainer,
) error {
	for gt, c := range chunkCont.Chunks {
		opts := ebiten.DrawImageOptions{}
		posX := float64(gt.X)*data.ChunkSize*data.TileSize - camera.X - data.TileSize/2
		posY := float64(gt.Y)*data.ChunkSize*data.TileSize - camera.Y - data.TileSize/2
		opts.GeoM.Translate(posX, posY)
		screen.DrawImage(c.Image, &opts)
	}

	return nil
}
