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

type SpriteManager struct{}

func NewSpriteComponent(imageUri string, layer uint8, allowRotation bool) (*sprite, error) {
	s := &sprite{offsetScale: 1, layer: layer, allowRotation: allowRotation}
	spr, img, err := ebitenutil.NewImageFromFile(imageUri)
	if err != nil {
		return nil, fmt.Errorf("failed to load sprite image: %w", err)
	}

	s.image = spr
	s.layerYOffset = utils.GetFirstOpaquePixelY(img)
	return s, nil
}

func (*SpriteManager) SetSpriteFlash(
	e common.EntityId,
	colors []utils.RelativeColor,
	colorDurationsMs []int,
	TotalDurationMs int,
	world *World,
) error {
	if len(colors) != len(colorDurationsMs) {
		return fmt.Errorf("colors and colorDurationsMs must have the same length")
	}

	sprite, err := world.Sprites.getComponent(e)
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

func (*SpriteManager) StopFlash(e common.EntityId, world *World) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.flash = nil

	return nil
}

func (*SpriteManager) GetImage(
	e common.EntityId,
	world *World,
) (*ebiten.Image, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return nil, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.image, nil
}

func (*SpriteManager) SetImage(
	e common.EntityId,
	image *ebiten.Image,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.image = image

	return nil
}

func (*SpriteManager) GetLocalOffsetPos(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetPos, nil
}

// TODO: Do these once for each entity in a transform system and cache them per tick.
// Unsure of performance benefit, but worth a try.
// Can use a 'dirty' flag and only recalculate entities that have moved
func (*SpriteManager) GetWorldOffsetPos(
	e common.EntityId,
	world *World,
) (utils.Vec2, error) {
	sprComp, err := world.Sprites.getComponent(e)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := TransformManager{}

	pWorldPos, err := tm.GetWorldPos(e, world)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	pWorldRot, err := tm.GetWorldRotation(e, world)
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

func (*SpriteManager) GetLocalOffsetRotation(
	e common.EntityId,
	world *World,
) (float64, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetRotation, nil
}

func (*SpriteManager) GetWorldOffsetRotation(
	e common.EntityId,
	world *World,
) (float64, error) {
	sprComp, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := TransformManager{}

	WorldRot, err := tm.GetWorldRotation(e, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", e, err)
	}

	return WorldRot + sprComp.offsetRotation, nil
}

func (*SpriteManager) GetLocalOffsetScale(
	e common.EntityId,
	world *World,
) (float64, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.offsetScale, nil
}

func (*SpriteManager) GetWorldOffsetScale(
	e common.EntityId,
	world *World,
) (float64, error) {
	pm := ParentManager{}

	sprComp, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	parEntity := pm.GetEntity(e, world)
	if parEntity == -1 {
		return sprComp.offsetScale, nil
	}

	tm := TransformManager{}

	pWorldSca, err := tm.GetWorldScale(parEntity, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	return pWorldSca * sprComp.offsetScale, nil
}

func (*SpriteManager) GetLocalLayerYOffset(
	e common.EntityId,
	world *World,
) (uint16, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.layerYOffset, nil
}

func (*SpriteManager) GetWorldLayerYOffset(
	e common.EntityId,
	world *World,
) (uint16, error) {
	sprComp, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	tm := TransformManager{}

	pWorldPos, err := tm.GetWorldPos(e, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	pWorldRot, err := tm.GetWorldRotation(e, world)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return uint16(pWorldPos.Y + (float64(sprComp.layerYOffset)*sin + float64(sprComp.layerYOffset)*cos)), nil
}

func (*SpriteManager) GetLayer(
	e common.EntityId,
	world *World,
) (uint8, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return 0, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.layer, nil
}

func (*SpriteManager) SetLocalOffsetPos(
	e common.EntityId,
	offset utils.Vec2,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetPos = offset

	return nil
}

func (*SpriteManager) SetLocalOffsetRotation(
	e common.EntityId,
	rotation float64,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetRotation = rotation

	return nil
}

func (*SpriteManager) SetLocalOffsetScale(
	e common.EntityId,
	scale float64,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.offsetScale = scale

	return nil
}

func (*SpriteManager) SetLocalLayerYOffset(
	e common.EntityId,
	offset uint16,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.layerYOffset = offset

	return nil
}

func (*SpriteManager) SetLayer(
	e common.EntityId,
	layer uint8,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.layer = layer

	return nil
}

func (*SpriteManager) GetAllowRotation(
	e common.EntityId,
	world *World,
) (bool, error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return false, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	return sprite.allowRotation, nil
}

func (*SpriteManager) SetAllowRotation(
	e common.EntityId,
	allow bool,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	sprite.allowRotation = allow

	return nil
}

func (*SpriteManager) GetCurrentColor(
	e common.EntityId,
	world *World,
) (color utils.RelativeColor, ok bool, err error) {
	sprite, err := world.Sprites.getComponent(e)
	if err != nil {
		return color, false, fmt.Errorf("could not get sprite of entity %d: %v", e, err)
	}

	if sprite.flash == nil {
		return color, false, nil
	}

	return sprite.flash.colors[sprite.flash.colorIdx], true, nil
}

func (*SpriteManager) TickFlash(
	e common.EntityId,
	world *World,
) error {
	sprite, err := world.Sprites.getComponent(e)
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
