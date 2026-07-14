package ecs

import (
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type spriteManager struct{}

func NewSpriteComponent(imageAssetTag common.ImageAssetTag, layer uint8, allowRotation bool) (*sprite, error) {
	s := &sprite{imageAssetTag: imageAssetTag, offsetScale: 1, layer: layer, allowRotation: allowRotation}
	return s, nil
}

func (*spriteManager) SetSpriteFlash(
	e common.EntityId,
	colors []utils.RelativeColor,
	colorDurationsMs []int,
	TotalDurationMs int,
	ecsContainer *ECSContainer,
) error {
	if len(colors) != len(colorDurationsMs) {
		return fmt.Errorf("colors and colorDurationsMs must have the same length")
	}

	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	var loopDurationMs int

	for _, d := range colorDurationsMs {
		loopDurationMs += d
	}

	f := SpriteFlash{
		Colors:           colors,
		ColorDurationsMs: colorDurationsMs,
		TotalDurationMs:  TotalDurationMs,
		LoopDurationMs:   loopDurationMs,
	}

	sprite.flash = &f

	return nil
}

func (*spriteManager) StopFlash(e common.EntityId, ecsContainer *ECSContainer) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.flash = nil

	return nil
}

func (*spriteManager) GetCurrentFrame(
	e common.EntityId,
	ecsContainer *ECSContainer,
	assetManager *assetmanager.AssetManager,
) (*ebiten.Image, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	imgAsset, err := assetManager.GetAsset(sprite.imageAssetTag)
	if err != nil {
		return nil, fmt.Errorf("could not get image asset for entity %d: %v", e, err)
	}

	if sprite.subImageIdx < 0 || sprite.subImageIdx >= len(imgAsset.Frames) {
		return nil, fmt.Errorf("subImageIdx %d out of bounds for image asset %s of entity %d", sprite.subImageIdx, sprite.imageAssetTag, e)
	}

	return imgAsset.Frames[sprite.subImageIdx], nil
}

func (*spriteManager) SetImageAsset(
	e common.EntityId,
	imageAssetTag common.ImageAssetTag,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.imageAssetTag = imageAssetTag

	return nil
}

func (*spriteManager) GetSubImageIdx(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (int, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.subImageIdx, nil
}

func (*spriteManager) SetSubImageIdx(
	e common.EntityId,
	subImageIdx int,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.subImageIdx = subImageIdx

	return nil
}

func (*spriteManager) GetLocalOffsetPos(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (utils.Vec2f, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetPos, nil
}

// TODO: Do these once for each entity in a transform system and cache them per tick.
// Unsure of performance benefit, but worth a try.
// Can use a 'dirty' flag and only recalculate entities that have moved
func (*spriteManager) GetWorldOffsetPos(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (utils.Vec2f, error) {
	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := transformManager{}

	pWorldPos, err := tm.GetWorldPos(e, ecsContainer)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	pWorldRot, err := tm.GetWorldRotation(e, ecsContainer)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2f{
		X: pWorldPos.X + (sprComp.offsetPos.X*cos - sprComp.offsetPos.Y*sin),
		Y: pWorldPos.Y + (sprComp.offsetPos.X*sin + sprComp.offsetPos.Y*cos),
	}, nil
}

func (*spriteManager) GetLocalOffsetRotation(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (float64, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetRotation, nil
}

func (*spriteManager) GetWorldOffsetRotation(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (float64, error) {
	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := transformManager{}

	WorldRot, err := tm.GetWorldRotation(e, ecsContainer)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", e, err)
	}

	return WorldRot + sprComp.offsetRotation, nil
}

func (*spriteManager) GetLocalOffsetScale(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (float64, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetScale, nil
}

func (*spriteManager) GetWorldOffsetScale(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (float64, error) {
	pm := parentManager{}

	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, ecsContainer)
	if parEntity == -1 {
		return sprComp.offsetScale, nil
	}

	tm := transformManager{}

	pWorldSca, err := tm.GetWorldScale(parEntity, ecsContainer)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	return pWorldSca * sprComp.offsetScale, nil
}

func (*spriteManager) GetLocalLayerYOffset(
	e common.EntityId,
	ecsContainer *ECSContainer,
	assetManager *assetmanager.AssetManager,
) (uint16, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	imgAsset, err := assetManager.GetAsset(sprite.imageAssetTag)
	if err != nil {
		return 0, fmt.Errorf("could not get image asset for entity %d: %v", e, err)
	}

	return imgAsset.LayerYOffset, nil
}

func (*spriteManager) GetWorldLayerYOffset(
	e common.EntityId,
	ecsContainer *ECSContainer,
	assetmanager *assetmanager.AssetManager,
) (uint16, error) {
	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := transformManager{}

	pWorldPos, err := tm.GetWorldPos(e, ecsContainer)
	if err != nil {
		return 0, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	pWorldRot, err := tm.GetWorldRotation(e, ecsContainer)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	imgAsset, err := assetmanager.GetAsset(sprComp.imageAssetTag)
	if err != nil {
		return 0, fmt.Errorf("could not get image asset for entity %d: %v", e, err)
	}

	return uint16(pWorldPos.Y + (float64(imgAsset.LayerYOffset)*sin + float64(imgAsset.LayerYOffset)*cos)), nil
}

func (*spriteManager) GetLayer(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (uint8, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.layer, nil
}

func (*spriteManager) SetLocalOffsetPos(
	e common.EntityId,
	offset utils.Vec2f,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetPos = offset

	return nil
}

func (*spriteManager) SetLocalOffsetRotation(
	e common.EntityId,
	rotation float64,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetRotation = rotation

	return nil
}

func (*spriteManager) SetLocalOffsetScale(
	e common.EntityId,
	scale float64,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetScale = scale

	return nil
}

func (*spriteManager) SetLayer(
	e common.EntityId,
	layer uint8,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.layer = layer

	return nil
}

func (*spriteManager) GetAllowRotation(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (bool, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.allowRotation, nil
}

func (*spriteManager) SetAllowRotation(
	e common.EntityId,
	allow bool,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.allowRotation = allow

	return nil
}

func (*spriteManager) GetCurrentColor(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (color utils.RelativeColor, ok bool, err error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return color, false, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	if sprite.flash == nil {
		return color, false, nil
	}

	return sprite.flash.Colors[sprite.flash.ColorIdx], true, nil
}

func (*spriteManager) TickFlash(
	e common.EntityId,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	if sprite.flash == nil {
		return nil
	}

	f := sprite.flash
	f.CounterMs += data.TickMs

	if f.CounterMs >= f.TotalDurationMs {
		sprite.flash = nil
	}

	colorCounterMs := f.CounterMs % f.LoopDurationMs

	for i, c := range f.ColorDurationsMs {
		if colorCounterMs < c {
			f.ColorIdx = i
			break
		}
		colorCounterMs -= c
	}

	return nil
}
