package assetmanager

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

type ImageAssetDef struct {
	path       string
	frameCount uint64
	frameSize  utils.Vec2f
}

var (
	assets = map[common.ImageAssetTag]ImageAssetDef{
		common.AssetSheetSlime:            ImageAssetDef{path: "assets/sprites/slime_ss.png", frameCount: 3, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetImageTree:             ImageAssetDef{path: "assets/sprites/tree.png", frameCount: 1, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetSheetEvilSlime:        ImageAssetDef{path: "assets/sprites/evilslime_ss.png", frameCount: 5, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetSheetBazooka:          ImageAssetDef{path: "assets/sprites/bazooka_ss.png", frameCount: 2, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetSheetRocket:           ImageAssetDef{path: "assets/sprites/rocket_ss.png", frameCount: 2, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetSheetExplosion:        ImageAssetDef{path: "assets/sprites/explosion_ss.png", frameCount: 7, frameSize: utils.Vec2f{X: 128, Y: 128}},
		common.AssetSheetGun:              ImageAssetDef{path: "assets/sprites/gun_ss.png", frameCount: 2, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetSheetBullet:           ImageAssetDef{path: "assets/sprites/bullet_ss.png", frameCount: 3, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetImagePlatform:         ImageAssetDef{path: "assets/sprites/platform.png", frameCount: 1, frameSize: utils.Vec2f{X: 32, Y: 32}},
		common.AssetTileImageDirt:         ImageAssetDef{path: "assets/tiles/dirt.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},
		common.AssetTileImageGrass:        ImageAssetDef{path: "assets/tiles/grass.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},
		common.AssetTileImageRock:         ImageAssetDef{path: "assets/tiles/rock.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},
		common.AssetTileImageShallowWater: ImageAssetDef{path: "assets/tiles/shallow_water.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},
		common.AssetTileImageDeepWater:    ImageAssetDef{path: "assets/tiles/deep_water.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},

		common.AssetDebug16x16: ImageAssetDef{path: "assets/debug/16x16.png", frameCount: 1, frameSize: utils.Vec2f{X: 16, Y: 16}},
	}
)
