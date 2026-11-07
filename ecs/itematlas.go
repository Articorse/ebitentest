package ecs

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs/common"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// What every generic item of a kind is, so that two of a kind are always the same and can stack without one turning
// into the other
type ItemDef struct {
	name       string
	imageAsset common.ImageAssetTag
	maxStack   int
	weight     int
	// Nil for items that do nothing in hand
	use *itemUse
	// Nil for a still image
	animation []AnimationFrame
	// 0 for items that give no light in hand
	lightLevel      uint8
	lightRangeTiles int
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

func (x ItemDef) Weight() int {
	return x.weight
}

type ItemAtlas struct {
	defs map[data.ItemEnum]ItemDef
}

type itemDefDTO struct {
	Id         uint16         `json:"id"`
	Name       string         `json:"name"`
	ImageAsset string         `json:"imageAsset"`
	MaxStack   int            `json:"maxStack"`
	Weight     int            `json:"weight"`
	Use        *itemUseDTO    `json:"use"`
	Animation  []itemFrameDTO `json:"animation"`
	Light      *itemLightDTO  `json:"light"`
}

type itemFrameDTO struct {
	FrameIdx   int `json:"frame"`
	DurationMs int `json:"durationMs"`
}

// Given while the item is in hand
type itemLightDTO struct {
	Level      uint8 `json:"level"`
	RangeTiles int   `json:"rangeTiles"`
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
		case dto.Weight < 0:
			return nil, fmt.Errorf("item %d weighs %d, which is negative", dto.Id, dto.Weight)
		}
		if _, ok := atlas.defs[kind]; ok {
			return nil, fmt.Errorf("item id %d is defined twice", dto.Id)
		}

		def := ItemDef{name: dto.Name, imageAsset: common.ImageAssetTag(dto.ImageAsset), maxStack: dto.MaxStack, weight: dto.Weight}
		if u := dto.Use; u != nil {
			if u.CooldownMs < 0 {
				return nil, fmt.Errorf("item %d has a negative cooldown", dto.Id)
			}
			def.use = &itemUse{ability: AbilityEnum(u.Ability), cooldownMs: u.CooldownMs, rawParams: u.Params}
		}
		for _, f := range dto.Animation {
			if f.FrameIdx < 0 || f.FrameIdx > math.MaxUint16 || f.DurationMs <= 0 {
				return nil, fmt.Errorf("item %d has an animation frame %d lasting %dms", dto.Id, f.FrameIdx, f.DurationMs)
			}
			def.animation = append(def.animation, AnimationFrame{FrameIdx: uint16(f.FrameIdx), DurationMs: uint64(f.DurationMs)})
		}
		if l := dto.Light; l != nil {
			if l.Level == 0 || l.RangeTiles <= 0 {
				return nil, fmt.Errorf("item %d gives light %d up to %d tiles away, which isn't positive", dto.Id, l.Level, l.RangeTiles)
			}
			def.lightLevel, def.lightRangeTiles = l.Level, l.RangeTiles
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

// Implemented by the params of abilities that place a feature
type FeaturePlacer interface {
	PlacedFeature() data.FeatureEnum
}

// The feature atlas is loaded after the item atlas, since features drop items, so the features that items place are
// checked once both are loaded
func (x *ItemAtlas) CheckPlacedFeatures(featureExists func(data.FeatureEnum) bool) error {
	for kind, def := range x.defs {
		if def.use == nil {
			continue
		}
		if placer, ok := def.use.params.(FeaturePlacer); ok && !featureExists(placer.PlacedFeature()) {
			return fmt.Errorf("item %d places feature %d, which isn't in the feature atlas", kind, placer.PlacedFeature())
		}
	}
	return nil
}

// Like the tiles and features they're drawn with, items name images by tag, and the frames their animations show
func (x *ItemAtlas) CheckImageAssets(assets *assetmanager.AssetManager) error {
	for kind, def := range x.defs {
		asset, err := assets.GetAsset(def.imageAsset)
		if err != nil {
			return fmt.Errorf("item %d's image: %w", kind, err)
		}
		for _, f := range def.animation {
			if int(f.FrameIdx) >= len(asset.Frames) {
				return fmt.Errorf("item %d's animation shows frame %d of %s, which has %d", kind, f.FrameIdx, def.imageAsset, len(asset.Frames))
			}
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
