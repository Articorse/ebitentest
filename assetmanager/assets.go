package assetmanager

import "ebittest/utils"

type ImageAssetTag string

type ImageAssetDef struct {
	path       string
	frameCount uint64
	frameSize  utils.Vec2
}

const (
	AssetSheetSlime            ImageAssetTag = "slime"
	AssetImageTree             ImageAssetTag = "tree"
	AssetSheetEvilSlime        ImageAssetTag = "evilslime"
	AssetSheetBazooka          ImageAssetTag = "bazooka"
	AssetSheetRocket           ImageAssetTag = "rocket"
	AssetSheetExplosion        ImageAssetTag = "explosion"
	AssetSheetGun              ImageAssetTag = "gun"
	AssetSheetBullet           ImageAssetTag = "bullet"
	AssetImagePlatform         ImageAssetTag = "platform"
	AssetTileImageDirt         ImageAssetTag = "tile_dirt"
	AssetTileImageGrass        ImageAssetTag = "tile_grass"
	AssetTileImageRock         ImageAssetTag = "tile_rock"
	AssetTileImageShallowWater ImageAssetTag = "tile_shallow_water"
	AssetTileImageDeepWater    ImageAssetTag = "tile_deep_water"

	AssetDebug16x16 ImageAssetTag = "debug_16x16"
)

var (
	assets = map[ImageAssetTag]ImageAssetDef{
		AssetSheetSlime:            ImageAssetDef{path: "assets/sprites/slime_ss.png", frameCount: 3, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetImageTree:             ImageAssetDef{path: "assets/sprites/tree.png", frameCount: 1, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetSheetEvilSlime:        ImageAssetDef{path: "assets/sprites/evilslime_ss.png", frameCount: 5, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetSheetBazooka:          ImageAssetDef{path: "assets/sprites/bazooka_ss.png", frameCount: 2, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetSheetRocket:           ImageAssetDef{path: "assets/sprites/rocket_ss.png", frameCount: 2, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetSheetExplosion:        ImageAssetDef{path: "assets/sprites/explosion_ss.png", frameCount: 7, frameSize: utils.Vec2{X: 128, Y: 128}},
		AssetSheetGun:              ImageAssetDef{path: "assets/sprites/gun_ss.png", frameCount: 2, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetSheetBullet:           ImageAssetDef{path: "assets/sprites/bullet_ss.png", frameCount: 3, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetImagePlatform:         ImageAssetDef{path: "assets/sprites/platform.png", frameCount: 1, frameSize: utils.Vec2{X: 32, Y: 32}},
		AssetTileImageDirt:         ImageAssetDef{path: "assets/tiles/dirt.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},
		AssetTileImageGrass:        ImageAssetDef{path: "assets/tiles/grass.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},
		AssetTileImageRock:         ImageAssetDef{path: "assets/tiles/rock.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},
		AssetTileImageShallowWater: ImageAssetDef{path: "assets/tiles/shallow_water.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},
		AssetTileImageDeepWater:    ImageAssetDef{path: "assets/tiles/deep_water.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},

		AssetDebug16x16: ImageAssetDef{path: "assets/debug/16x16.png", frameCount: 1, frameSize: utils.Vec2{X: 16, Y: 16}},
	}
)
