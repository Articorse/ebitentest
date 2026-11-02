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
	// Nil for items that do nothing in hand
	use *itemUse
}

// The ability a generic item has in hand, which is decoded once the abilities it can name are registered
type itemUse struct {
	ability    AbilityEnum
	cooldownMs int
	rawParams  json.RawMessage
	params     AbilityParams
}

// Turns an ability's params, as the item atlas has them, into the params its ability function expects
type AbilityParamsDecoder func(raw json.RawMessage) (AbilityParams, error)

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
	Id         uint16      `json:"id"`
	Name       string      `json:"name"`
	ImageAsset string      `json:"imageAsset"`
	MaxStack   int         `json:"maxStack"`
	Use        *itemUseDTO `json:"use"`
}

type itemUseDTO struct {
	Ability    uint64          `json:"ability"`
	CooldownMs int             `json:"cooldownMs"`
	Params     json.RawMessage `json:"params"`
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

		def := ItemDef{name: dto.Name, imageAsset: common.ImageAssetTag(dto.ImageAsset), maxStack: dto.MaxStack}
		if u := dto.Use; u != nil {
			if u.CooldownMs < 0 {
				return nil, fmt.Errorf("item %d has a negative cooldown", dto.Id)
			}
			def.use = &itemUse{ability: AbilityEnum(u.Ability), cooldownMs: u.CooldownMs, rawParams: u.Params}
		}
		atlas.defs[kind] = def
	}

	return atlas, nil
}

func (x *ItemAtlas) Def(kind data.ItemEnum) (ItemDef, bool) {
	def, ok := x.defs[kind]
	return def, ok
}

// Content has to be registered first, since the atlas's items name its abilities, whose params are decoded here
func (x *ECSContainer) SetItemAtlas(atlas *ItemAtlas) error {
	for kind, def := range atlas.defs {
		if def.use == nil {
			continue
		}

		if f, err := x.AbilityFunc(def.use.ability); err != nil || f == nil {
			return fmt.Errorf("item %d uses ability %v, which isn't registered", kind, def.use.ability)
		}

		decode, ok := x.abilityParamsDecoders[def.use.ability]
		if !ok {
			return fmt.Errorf("item %d uses ability %v, whose params can't be given by the item atlas", kind, def.use.ability)
		}
		params, err := decode(def.use.rawParams)
		if err != nil {
			return fmt.Errorf("error decoding the params of item %d's ability: %w", kind, err)
		}
		def.use.params = params
	}

	x.itemAtlas = atlas
	return nil
}

// Implemented by the params of abilities that place a tile
type TilePlacer interface {
	PlacedTile() data.TileEnum
}

// The tile atlas is loaded after the item atlas, since tiles drop items, so the tiles that items place are checked
// once both are loaded
func (x *ItemAtlas) CheckPlacedTiles(tileExists func(data.TileEnum) bool) error {
	for kind, def := range x.defs {
		if def.use == nil {
			continue
		}
		if placer, ok := def.use.params.(TilePlacer); ok && !tileExists(placer.PlacedTile()) {
			return fmt.Errorf("item %d places tile %d, which isn't in the tile atlas", kind, placer.PlacedTile())
		}
	}
	return nil
}

// What a generic item of the kind holds in hand, if anything
func (x *ECSContainer) itemEquipment(def ItemDef) (*Equipment, bool) {
	if def.use == nil {
		return nil, false
	}

	abis := [data.MaxEquipmentAbilitySlots]EntityAbility{{
		Def:    NewAbilityDef(def.use.ability, Ability_None, def.use.cooldownMs, 0),
		Params: def.use.params.Copy(),
		Status: AbilityStatus{State: AbiAct_Ready},
	}}
	return NewEquipmentComponent(Equipable_MainHand|Equipable_OffHand, abis), true
}

func (x *ECSContainer) ItemAtlas() *ItemAtlas {
	return x.itemAtlas
}
