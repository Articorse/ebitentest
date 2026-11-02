package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"encoding/json"
	"fmt"
	"os"
)

// What every generic item of a kind is, so that two of a kind are always the same and can stack without one turning
// into the other
type ItemDef struct {
	name       string
	imageAsset common.ImageAssetTag
	maxStack   int
}

func (x ItemDef) Name() string {
	return x.name
}

func (x ItemDef) ImageAsset() common.ImageAssetTag {
	return x.imageAsset
}

func (x ItemDef) MaxStack() int {
	return x.maxStack
}

type ItemAtlas struct {
	defs map[data.ItemEnum]ItemDef
}

type itemDefDTO struct {
	Id         uint16 `json:"id"`
	Name       string `json:"name"`
	ImageAsset string `json:"imageAsset"`
	MaxStack   int    `json:"maxStack"`
}

func NewItemAtlas() *ItemAtlas {
	return &ItemAtlas{defs: map[data.ItemEnum]ItemDef{}}
}

func LoadItemAtlasFromJson(path string) (*ItemAtlas, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading item atlas json file: %w", err)
	}

	var dtos []itemDefDTO
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling item atlas json: %w", err)
	}

	atlas := NewItemAtlas()
	for _, dto := range dtos {
		kind := data.ItemEnum(dto.Id)
		switch {
		case kind == data.Item_None:
			return nil, fmt.Errorf("item %q has id %d, which is kept for unique items", dto.Name, dto.Id)
		case dto.Name == "" || dto.ImageAsset == "":
			return nil, fmt.Errorf("item %d needs a name and an image asset", dto.Id)
		case dto.MaxStack < 1:
			return nil, fmt.Errorf("item %d stacks up to %d, want at least 1", dto.Id, dto.MaxStack)
		}
		if _, ok := atlas.defs[kind]; ok {
			return nil, fmt.Errorf("item id %d is defined twice", dto.Id)
		}

		atlas.defs[kind] = ItemDef{name: dto.Name, imageAsset: common.ImageAssetTag(dto.ImageAsset), maxStack: dto.MaxStack}
	}

	return atlas, nil
}

func (x *ItemAtlas) Def(kind data.ItemEnum) (ItemDef, bool) {
	def, ok := x.defs[kind]
	return def, ok
}

func (x *ECSContainer) SetItemAtlas(atlas *ItemAtlas) {
	x.itemAtlas = atlas
}

func (x *ECSContainer) ItemAtlas() *ItemAtlas {
	return x.itemAtlas
}
