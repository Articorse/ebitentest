package tilesystem

import (
	"ebittest/data"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type tileDef struct {
	image     *ebiten.Image
	maxHealth int
	passable  bool
	opaque    bool // TODO: Use this for a transparency system
}

type promotedTile struct {
	currentHealth int
}

type tileDefDTO struct {
	Id        uint16 `json:"id"`
	ImagePath string `json:"imagePath"`
	MaxHealth int    `json:"maxHealth"`
	Passable  bool   `json:"passable"`
	Opaque    bool   `json:"opaque"`
}

func dtosToDefsMap(dtos []tileDefDTO) (map[data.TileEnum]tileDef, error) {
	tileDefs := make(map[data.TileEnum]tileDef, len(dtos))
	for _, dto := range dtos {
		img, _, err := ebitenutil.NewImageFromFile(dto.ImagePath)
		if err != nil {
			log.Printf("Error loading tile image from path %s: %v\n", dto.ImagePath, err)
			img, _, err = ebitenutil.NewImageFromFile("assets/debug/16x16.png")
			if err != nil {
				return nil, fmt.Errorf("error loading fallback tile image: %v", err)
			}
		}

		tileDefs[data.TileEnum(dto.Id)] = tileDef{
			image:     img,
			maxHealth: dto.MaxHealth,
			passable:  dto.Passable,
			opaque:    dto.Opaque,
		}
	}

	return tileDefs, nil
}
