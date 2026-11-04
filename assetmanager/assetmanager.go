package assetmanager

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type ImageAsset struct {
	Frames       []*ebiten.Image
	FrameSize    utils.Vec2f
	LayerYOffset uint16
}

type AssetManager struct {
	StaticAssets map[common.ImageAssetTag]*ImageAsset
}

type imageAssetDefDTO struct {
	Tag string `json:"tag"`
	// Relative to the asset list's own folder, so that it loads the same from wherever the game is run
	Path       string      `json:"path"`
	FrameCount int         `json:"frameCount"`
	FrameSize  utils.Vec2f `json:"frameSize"`
}

// Every image the game draws, by the tag that code and the other atlases name it by. An image that's missing is drawn
// as the debug image instead, so that a missing file shows rather than stopping the game, but one that's there has to
// be exactly as wide and tall as its frames say.
func LoadAssetManagerFromJson(path string) (*AssetManager, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading asset list json file: %w", err)
	}

	var dtos []imageAssetDefDTO
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling asset list json: %w", err)
	}

	debugIdx := slices.IndexFunc(dtos, func(d imageAssetDefDTO) bool { return common.ImageAssetTag(d.Tag) == common.AssetDebug16x16 })
	if debugIdx == -1 {
		return nil, fmt.Errorf("the asset list has no %s, which missing images are drawn as", common.AssetDebug16x16)
	}
	dir := filepath.Dir(path)
	debug, err := loadImage(dir, dtos[debugIdx], nil)
	if err != nil {
		return nil, fmt.Errorf("error loading the debug image: %w", err)
	}

	assets := make(map[common.ImageAssetTag]*ImageAsset, len(dtos))
	for _, dto := range dtos {
		tag := common.ImageAssetTag(dto.Tag)
		switch _, taken := assets[tag]; {
		case tag == "" || dto.Path == "":
			return nil, fmt.Errorf("an asset needs a tag and a path, got %q and %q", dto.Tag, dto.Path)
		case taken:
			return nil, fmt.Errorf("asset %s is defined twice", tag)
		case dto.FrameCount < 1 || dto.FrameSize.X < 1 || dto.FrameSize.Y < 1:
			return nil, fmt.Errorf("asset %s has %d frames of %v×%v, want at least one frame of at least a pixel", tag, dto.FrameCount, dto.FrameSize.X, dto.FrameSize.Y)
		}

		if tag == common.AssetDebug16x16 {
			assets[tag] = debug
			continue
		}
		img, err := loadImage(dir, dto, debug)
		if err != nil {
			return nil, fmt.Errorf("error loading asset %s: %w", tag, err)
		}
		assets[tag] = img
	}

	return &AssetManager{StaticAssets: assets}, nil
}

// Nil fallback for an image that has to be there
func loadImage(dir string, dto imageAssetDefDTO, fallback *ImageAsset) (*ImageAsset, error) {
	path := filepath.Join(dir, dto.Path)
	img, rawImg, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		if fallback == nil {
			return nil, err
		}
		log.Printf("Failed to load image from %s, drawing the debug image instead: %v\n", path, err)
		frames := make([]*ebiten.Image, dto.FrameCount)
		for i := range frames {
			frames[i] = fallback.Frames[0]
		}
		return &ImageAsset{Frames: frames, FrameSize: fallback.FrameSize, LayerYOffset: fallback.LayerYOffset}, nil
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	frameW, frameH := int(dto.FrameSize.X), int(dto.FrameSize.Y)
	if w != dto.FrameCount*frameW || h != frameH {
		return nil, fmt.Errorf("image %s is %d×%d, want %d frames of %d×%d side by side", path, w, h, dto.FrameCount, frameW, frameH)
	}

	frames := make([]*ebiten.Image, dto.FrameCount)
	for i := range frames {
		frames[i] = img.SubImage(image.Rect(i*frameW, 0, (i+1)*frameW, frameH)).(*ebiten.Image)
	}

	return &ImageAsset{
		Frames:       frames,
		FrameSize:    dto.FrameSize,
		LayerYOffset: utils.GetFirstOpaquePixelY(rawImg),
	}, nil
}

func (x *AssetManager) GetAsset(assetName common.ImageAssetTag) (*ImageAsset, error) {
	asset, ok := x.StaticAssets[assetName]
	if !ok {
		return nil, fmt.Errorf("asset %s not found", assetName)
	}

	return asset, nil
}

func (x *AssetManager) HasAsset(tag common.ImageAssetTag) bool {
	_, ok := x.StaticAssets[tag]
	return ok
}

// So that a tag the asset list lost fails at startup rather than when it's first drawn
func (x *AssetManager) CheckTags(tags []common.ImageAssetTag) error {
	for _, tag := range tags {
		if !x.HasAsset(tag) {
			return fmt.Errorf("asset %s is used, but isn't in the asset list", tag)
		}
	}
	return nil
}
