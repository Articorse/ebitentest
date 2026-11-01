package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type tileDef struct {
	image *ebiten.Image
	// Not positive for tiles that can't be damaged
	maxHealth     int
	destroyedInto data.TileEnum
	passable      bool
	opaque        bool
}

type promotedTile struct {
	currentHealth int
}

type tileDefDTO struct {
	Id        uint16 `json:"id"`
	ImagePath string `json:"imagePath"`
	MaxHealth int    `json:"maxHealth"`
	// Required for tiles that can be damaged
	DestroyedInto *uint16 `json:"destroyedInto"`
	Passable      bool    `json:"passable"`
	Opaque        bool    `json:"opaque"`
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

		def := tileDef{
			image:     img,
			maxHealth: dto.MaxHealth,
			passable:  dto.Passable,
			opaque:    dto.Opaque,
		}

		if def.maxHealth > 0 {
			if dto.DestroyedInto == nil {
				return nil, fmt.Errorf("tile %d can be damaged, but doesn't say what it's destroyed into", dto.Id)
			}
			def.destroyedInto = data.TileEnum(*dto.DestroyedInto)
		}

		tileDefs[data.TileEnum(dto.Id)] = def
	}

	for id, def := range tileDefs {
		if _, ok := tileDefs[def.destroyedInto]; def.maxHealth > 0 && !ok {
			return nil, fmt.Errorf("tile %d is destroyed into tile %d, which isn't defined", id, def.destroyedInto)
		}
	}

	return tileDefs, nil
}

// Tiles outside the active chunks count as opaque, since nothing about them is known to the simulation
func (cc *ChunkContainer) IsOpaqueTile(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return true
	}

	def, ok := cc.atlas[c.getTileDefId(t)]
	return !ok || def.opaque
}
