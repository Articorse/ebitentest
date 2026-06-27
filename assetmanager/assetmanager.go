package assetmanager

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type ImageAsset struct {
	Frames       []*ebiten.Image
	FrameSize    utils.Vec2
	LayerYOffset uint16
}

type AssetManager struct {
	StaticAssets map[common.ImageAssetTag]*ImageAsset
}

func NewAssetManager() *AssetManager {
	a := make(map[common.ImageAssetTag]*ImageAsset, len(assets))
	for assetName, asset := range assets {
		img, err := loadImage(asset)
		if err != nil {
			log.Fatalf("failed to load asset %s from '%s': %v", assetName, asset.path, err)
		}
		a[assetName] = img
	}

	return &AssetManager{
		StaticAssets: a,
	}
}

func loadImage(def ImageAssetDef) (*ImageAsset, error) {
	img, rawImg, err := ebitenutil.NewImageFromFile(def.path)
	if err != nil {
		fmt.Printf("Failed to load image from %s: %v\n", def.path, err)
		img, rawImg, err = ebitenutil.NewImageFromFile(assets[common.AssetDebug16x16].path)
		if err != nil {
			return nil, fmt.Errorf("failed to load debug image from %s: %w", common.AssetDebug16x16, err)
		}
	}

	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return nil, fmt.Errorf("image %s has invalid dimensions: %dx%d", def.path, img.Bounds().Dx(), img.Bounds().Dy())
	}

	if img.Bounds().Dy() < int(def.frameSize.Y) {
		return nil, fmt.Errorf("image %s height %d is less than frame size %f", def.path, img.Bounds().Dy(), def.frameSize.Y)
	}

	if img.Bounds().Dx()%int(def.frameSize.X) != 0 {
		return nil, fmt.Errorf("image %s width %d is not a multiple of frame size %f", def.path, img.Bounds().Dx(), def.frameSize.X)
	}

	var frames []*ebiten.Image
	for x := 0; x < img.Bounds().Dx(); x += int(def.frameSize.X) {
		frame := img.SubImage(image.Rect(x, 0, x+int(def.frameSize.X), int(def.frameSize.Y))).(*ebiten.Image)
		frames = append(frames, frame)
	}

	return &ImageAsset{
		Frames:       frames,
		FrameSize:    def.frameSize,
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
