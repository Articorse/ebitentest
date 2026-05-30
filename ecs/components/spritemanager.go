package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type SpriteManager struct{}

func NewSpriteComponent(imageUri string, layer uint8) (*Sprite, error) {
	s := &Sprite{offsetScale: 1, layer: layer}
	spr, img, err := ebitenutil.NewImageFromFile(imageUri)
	if err != nil {
		return nil, fmt.Errorf("failed to load sprite image: %w", err)
	}

	s.image = spr
	s.layerYOffset = utils.GetFirstOpaquePixelY(img)
	return s, nil
}

func (*SpriteManager) GetImage(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (*ebiten.Image, error) {
	sprite, ok := sprites[e]
	if !ok {
		return nil, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.image, nil
}

func (*SpriteManager) SetImage(
	e ecscommon.EntityId,
	image *ebiten.Image,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.image = image

	return nil
}

func (*SpriteManager) GetLocalOffsetPos(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (utils.Vec2, error) {
	sprite, ok := sprites[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.offsetPos, nil
}

func (*SpriteManager) GetWorldOffsetPos(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (utils.Vec2, error) {
	sprComp, ok := sprites[e]
	if !ok {
		return utils.Vec2{}, fmt.Errorf("could not get sprite of entity %d", e)
	}

	tm := TransformManager{}

	pWorldPos, err := tm.GetWorldPos(e, transforms, parents)
	if err != nil {
		return utils.Vec2{}, fmt.Errorf("error getting world position of entity %d: %v", e, ok)
	}

	pWorldRot, err := tm.GetWorldRotation(e, transforms, parents)
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
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (float64, error) {
	sprite, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.offsetRotation, nil
}

func (*SpriteManager) GetWorldOffsetRotation(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (float64, error) {
	pm := ParentManager{}

	sprComp, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return sprComp.offsetRotation, nil
	}

	tm := TransformManager{}

	pWorldRot, err := tm.GetWorldRotation(parEntity, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of parent entity %d: %v", parEntity, err)
	}

	return pWorldRot + sprComp.offsetRotation, nil
}

func (*SpriteManager) GetLocalOffsetScale(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (float64, error) {
	sprite, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.offsetScale, nil
}

func (*SpriteManager) GetWorldOffsetScale(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (float64, error) {
	pm := ParentManager{}

	sprComp, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	parEntity := pm.GetEntity(e, parents)
	if parEntity == -1 {
		return sprComp.offsetScale, nil
	}

	tm := TransformManager{}

	pWorldSca, err := tm.GetWorldScale(parEntity, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", parEntity, err)
	}

	return pWorldSca * sprComp.offsetScale, nil
}

func (*SpriteManager) GetLocalLayerYOffset(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (uint16, error) {
	sprite, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.layerYOffset, nil
}

func (*SpriteManager) GetWorldLayerYOffset(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) (uint16, error) {
	sprComp, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	tm := TransformManager{}

	pWorldPos, err := tm.GetWorldPos(e, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world position of entity %d: %v", e, ok)
	}

	pWorldRot, err := tm.GetWorldRotation(e, transforms, parents)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	return uint16(pWorldPos.Y + (float64(sprComp.layerYOffset)*sin + float64(sprComp.layerYOffset)*cos)), nil
}

func (*SpriteManager) GetLayer(
	e ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*Sprite,
) (uint8, error) {
	sprite, ok := sprites[e]
	if !ok {
		return 0, fmt.Errorf("could not get sprite of entity %d", e)
	}

	return sprite.layer, nil
}

func (*SpriteManager) SetLocalOffsetPos(
	e ecscommon.EntityId,
	offset utils.Vec2,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.offsetPos = offset

	return nil
}

func (*SpriteManager) SetLocalOffsetRotation(
	e ecscommon.EntityId,
	rotation float64,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.offsetRotation = rotation

	return nil
}

func (*SpriteManager) SetLocalOffsetScale(
	e ecscommon.EntityId,
	scale float64,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.offsetScale = scale

	return nil
}

func (*SpriteManager) SetLocalLayerYOffset(
	e ecscommon.EntityId,
	offset uint16,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.layerYOffset = offset

	return nil
}

func (*SpriteManager) SetLayer(
	e ecscommon.EntityId,
	layer uint8,
	sprites map[ecscommon.EntityId]*Sprite,
) error {
	sprite, ok := sprites[e]
	if !ok {
		return fmt.Errorf("could not get sprite of entity %d", e)
	}

	sprite.layer = layer

	return nil
}
