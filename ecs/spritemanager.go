package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type spriteManager struct{}

func NewSpriteComponent(imageUri string, layer uint8, allowRotation bool) (*sprite, error) {
	s := &sprite{offsetScale: 1, layer: layer, allowRotation: allowRotation}

	if len(imageUri) > 0 {
		spr, img, err := ebitenutil.NewImageFromFile(imageUri)
		if err != nil {
			return nil, fmt.Errorf("failed to load sprite image: %w", err)
		}

		s.image = spr
		s.layerYOffset = utils.GetFirstOpaquePixelY(img)
	}
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
		colors:           colors,
		colorDurationsMs: colorDurationsMs,
		totalDurationMs:  TotalDurationMs,
		loopDurationMs:   loopDurationMs,
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

func (*spriteManager) GetImage(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (*ebiten.Image, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.image, nil
}

func (*spriteManager) SetImage(
	e common.EntityId,
	image *ebiten.Image,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.image = image

	return nil
}

func (*spriteManager) GetLocalOffsetPos(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (utils.Vec2, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetPos, nil
}

// TODO: Do these once for each entity in a transform system and cache them per tick.
// Unsure of performance benefit, but worth a try.
// Can use a 'dirty' flag and only recalculate entities that have moved
func (*spriteManager) GetWorldOffsetPos(
	e common.EntityId,
	ecsContainer *ECSContainer,
) (utils.Vec2, error) {
	sprComp, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := transformManager{}

	pWorldPos, err := tm.GetWorldPos(e, ecsContainer)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	pWorldRot, err := tm.GetWorldRotation(e, ecsContainer)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return utils.Vec2{
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
) (uint16, error) {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.layerYOffset, nil
}

func (*spriteManager) GetWorldLayerYOffset(
	e common.EntityId,
	ecsContainer *ECSContainer,
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

	return uint16(pWorldPos.Y + (float64(sprComp.layerYOffset)*sin + float64(sprComp.layerYOffset)*cos)), nil
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
	offset utils.Vec2,
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

func (*spriteManager) SetLocalLayerYOffset(
	e common.EntityId,
	offset uint16,
	ecsContainer *ECSContainer,
) error {
	sprite, err := ecsContainer.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.layerYOffset = offset

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

	return sprite.flash.colors[sprite.flash.colorIdx], true, nil
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
	f.counterMs += data.TickMs

	if f.counterMs >= f.totalDurationMs {
		sprite.flash = nil
	}

	colorCounterMs := f.counterMs % f.loopDurationMs

	for i, c := range f.colorDurationsMs {
		if colorCounterMs < c {
			f.colorIdx = i
			break
		}
		colorCounterMs -= c
	}

	return nil
}
