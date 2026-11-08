package ecs

import (
	"cmp"
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs/common"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
)

// A tile or feature that's built rather than placed: a blueprint is put down, the materials it costs are brought to it,
// and it's worked on until it's the tile or feature
type BlueprintDef struct {
	name string
	// Tile_None for a blueprint that builds a feature, and Feature_None for one that builds a tile
	tile    data.TileEnum
	feature data.FeatureEnum
	// Its first frame is the finished tile, and the rest are the stages of building it, in order
	imageAsset common.ImageAssetTag
	stages     int
	costs      []ItemCost
	// How much building the tile takes, so that what builds faster gets it done in fewer strikes
	work int
	// What it looks like before it's built, in the menu and as a blueprint
	ghostAsset common.ImageAssetTag
	ghostFrame int
}

type ItemCost struct {
	Kind  data.ItemEnum
	Count int
}

func (x BlueprintDef) Name() string {
	return x.name
}

func (x BlueprintDef) Tile() data.TileEnum {
	return x.tile
}

func (x BlueprintDef) Feature() data.FeatureEnum {
	return x.feature
}

func (x BlueprintDef) ImageAsset() common.ImageAssetTag {
	return x.imageAsset
}

func (x BlueprintDef) Ghost() (common.ImageAssetTag, int) {
	return x.ghostAsset, x.ghostFrame
}

// The frames after the first
func (x BlueprintDef) Stages() int {
	return x.stages
}

func (x BlueprintDef) Costs() []ItemCost {
	return slices.Clone(x.costs)
}

func (x BlueprintDef) Work() int {
	return x.work
}

type BlueprintAtlas struct {
	defs map[data.BlueprintEnum]BlueprintDef
}

type blueprintDefDTO struct {
	Id         uint16        `json:"id"`
	Name       string        `json:"name"`
	Tile       uint16        `json:"tile"`
	ImageAsset string        `json:"imageAsset"`
	Costs      []itemCostDTO `json:"costs"`
	Work       int           `json:"work"`
	Feature    uint16        `json:"feature"`
	// The first frame of imageAsset when it's not given
	Ghost *ghostDTO `json:"ghost"`
}

type ghostDTO struct {
	Asset string `json:"asset"`
	Frame int    `json:"frame"`
}

type itemCostDTO struct {
	Kind  uint16 `json:"kind"`
	Count int    `json:"count"`
}

func NewBlueprintAtlas() *BlueprintAtlas {
	return &BlueprintAtlas{defs: map[data.BlueprintEnum]BlueprintDef{}}
}

// Loaded after the item atlas, since blueprints cost items. The tiles they build are checked when the terrain is given
// the atlas.
func LoadBlueprintAtlasFromJson(path string, items *ItemAtlas, assets *assetmanager.AssetManager) (*BlueprintAtlas, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading blueprint atlas json file: %w", err)
	}

	var dtos []blueprintDefDTO
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling blueprint atlas json: %w", err)
	}

	atlas := NewBlueprintAtlas()
	for _, dto := range dtos {
		kind := data.BlueprintEnum(dto.Id)
		switch {
		case kind == data.Blueprint_None:
			return nil, fmt.Errorf("blueprint %q has id 0, which means none", dto.Name)
		case dto.Name == "":
			return nil, fmt.Errorf("blueprint %d needs a name", dto.Id)
		case dto.Work <= 0:
			return nil, fmt.Errorf("blueprint %d takes %d work, which isn't positive", dto.Id, dto.Work)
		case len(dto.Costs) == 0:
			return nil, fmt.Errorf("blueprint %d costs nothing", dto.Id)
		case (dto.Tile == 0) == (dto.Feature == 0):
			return nil, fmt.Errorf("blueprint %d builds tile %d and feature %d, want one or the other", dto.Id, dto.Tile, dto.Feature)
		}
		if _, ok := atlas.defs[kind]; ok {
			return nil, fmt.Errorf("blueprint id %d is defined twice", dto.Id)
		}

		asset, err := assets.GetAsset(common.ImageAssetTag(dto.ImageAsset))
		if err != nil {
			return nil, fmt.Errorf("blueprint %d's image: %w", dto.Id, err)
		}
		if len(asset.Frames) < 2 {
			return nil, fmt.Errorf("blueprint %d's image has %d frames, want the finished tile and at least one stage", dto.Id, len(asset.Frames))
		}

		def := BlueprintDef{
			name:       dto.Name,
			tile:       data.TileEnum(dto.Tile),
			feature:    data.FeatureEnum(dto.Feature),
			imageAsset: common.ImageAssetTag(dto.ImageAsset),
			stages:     len(asset.Frames) - 1,
			work:       dto.Work,
			ghostAsset: common.ImageAssetTag(dto.ImageAsset),
		}
		if g := dto.Ghost; g != nil {
			ghost, err := assets.GetAsset(common.ImageAssetTag(g.Asset))
			if err != nil {
				return nil, fmt.Errorf("blueprint %d's ghost: %w", dto.Id, err)
			}
			if g.Frame < 0 || g.Frame >= len(ghost.Frames) {
				return nil, fmt.Errorf("blueprint %d's ghost is frame %d of %d", dto.Id, g.Frame, len(ghost.Frames))
			}
			def.ghostAsset, def.ghostFrame = common.ImageAssetTag(g.Asset), g.Frame
		}
		for _, c := range dto.Costs {
			cost := ItemCost{Kind: data.ItemEnum(c.Kind), Count: c.Count}
			if _, ok := items.Def(cost.Kind); !ok {
				return nil, fmt.Errorf("blueprint %d costs item %d, which isn't in the item atlas", dto.Id, c.Kind)
			}
			if cost.Count <= 0 || slices.ContainsFunc(def.costs, func(o ItemCost) bool { return o.Kind == cost.Kind }) {
				return nil, fmt.Errorf("blueprint %d costs %d of item %d, want a positive count once per item", dto.Id, c.Count, c.Kind)
			}
			def.costs = append(def.costs, cost)
		}
		atlas.defs[kind] = def
	}

	return atlas, nil
}

func (x *BlueprintAtlas) Def(kind data.BlueprintEnum) (BlueprintDef, bool) {
	def, ok := x.defs[kind]
	return def, ok
}

// In id order, the order they're shown in
func (x *BlueprintAtlas) Kinds() []data.BlueprintEnum {
	return slices.SortedFunc(maps.Keys(x.defs), cmp.Compare)
}

// The first that builds the tile, in the order they're shown in
func (x *BlueprintAtlas) ForTile(tile data.TileEnum) (data.BlueprintEnum, bool) {
	for _, kind := range x.Kinds() {
		if x.defs[kind].tile == tile {
			return kind, true
		}
	}
	return data.Blueprint_None, false
}

func (x *ECSContainer) SetBlueprintAtlas(atlas *BlueprintAtlas) {
	x.blueprintAtlas = atlas
}

func (x *ECSContainer) BlueprintAtlas() *BlueprintAtlas {
	return x.blueprintAtlas
}
