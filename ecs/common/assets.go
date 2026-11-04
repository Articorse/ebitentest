package common

// Names an image in the asset list, assets/assets.json, which is the only place that knows its file
type ImageAssetTag string

const (
	AssetSheetSlime         ImageAssetTag = "slime"
	AssetImageTree          ImageAssetTag = "tree"
	AssetImageTreeStump     ImageAssetTag = "treestump"
	AssetSheetEvilSlime     ImageAssetTag = "evilslime"
	AssetSheetBazooka       ImageAssetTag = "bazooka"
	AssetSheetRocket        ImageAssetTag = "rocket"
	AssetSheetExplosion     ImageAssetTag = "explosion"
	AssetSheetGun           ImageAssetTag = "gun"
	AssetSheetBullet        ImageAssetTag = "bullet"
	AssetImageMattock       ImageAssetTag = "mattock"
	AssetImageWood          ImageAssetTag = "wood"
	AssetSheetTorchPlaced   ImageAssetTag = "torch_placed"
	AssetImageAxe           ImageAssetTag = "axe"
	AssetImageSlash         ImageAssetTag = "slash"
	AssetImagePlatform      ImageAssetTag = "platform"
	AssetImageSlot          ImageAssetTag = "slot"
	AssetImageSlotHighlight ImageAssetTag = "slot_highlight"
	// Nine slices of three by three pixels: corners, edges and middle
	AssetImageInventoryPanel ImageAssetTag = "inventory_panel"

	AssetDebug16x16 ImageAssetTag = "debug_16x16"
)

// The tags code names, which the asset list is checked against at startup. Atlases name theirs as data, and are checked
// when they're loaded.
var CodeAssetTags = []ImageAssetTag{
	AssetSheetSlime, AssetImageTree, AssetImageTreeStump, AssetSheetEvilSlime, AssetSheetBazooka, AssetSheetRocket,
	AssetSheetExplosion, AssetSheetGun, AssetSheetBullet, AssetImageMattock, AssetImageWood, AssetSheetTorchPlaced,
	AssetImageAxe, AssetImageSlash, AssetImagePlatform, AssetImageSlot, AssetImageSlotHighlight,
	AssetImageInventoryPanel, AssetDebug16x16,
}
