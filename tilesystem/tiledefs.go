package tilesystem

import (
	"ebittest/data"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type TileDef struct {
	Image     *ebiten.Image
	MaxHealth int
	Passable  bool
	Opaque    bool // TODO: Use this for a visibility system
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

func dtosToDefsMap(dtos []tileDefDTO) (map[data.TileEnum]TileDef, error) {
	tileDefs := make(map[data.TileEnum]TileDef, len(dtos))
	for _, dto := range dtos {
		img, _, err := ebitenutil.NewImageFromFile(dto.ImagePath)
		if err != nil {
			log.Printf("Error loading tile image from path %s: %v\n", dto.ImagePath, err)
			img, _, err = ebitenutil.NewImageFromFile("assets/debug/16x16.png")
			if err != nil {
				return nil, fmt.Errorf("error loading fallback tile image: %v", err)
			}
		}

		tileDefs[data.TileEnum(dto.Id)] = TileDef{
			Image:     img,
			MaxHealth: dto.MaxHealth,
			Passable:  dto.Passable,
			Opaque:    dto.Opaque,
		}
	}

	return tileDefs, nil
}
