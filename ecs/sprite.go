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

type SpriteFlash struct {
	Colors           []utils.RelativeColor
	ColorDurationsMs []int
	TotalDurationMs  int

	ColorIdx       int
	CounterMs      int
	LoopDurationMs int
}

type Sprite struct {
	imageAssetTag  common.ImageAssetTag
	subImageIdx    int
	offsetPos      utils.Vec2f
	offsetScale    float64
	offsetRotation float64
	layer          uint8
	allowRotation  bool
	flash          *SpriteFlash
}

func NewSpriteComponent(imageAssetTag common.ImageAssetTag, layer uint8, allowRotation bool) (*Sprite, error) {
	s := &Sprite{imageAssetTag: imageAssetTag, offsetScale: 1, layer: layer, allowRotation: allowRotation}
	return s, nil
}

func (x *Sprite) SetFlash(colors []utils.RelativeColor, colorDurationsMs []int, totalDurationMs int) error {
	if len(colors) != len(colorDurationsMs) {
		return fmt.Errorf("colors and colorDurationsMs must have the same length")
	}

	var loopDurationMs int
	for _, d := range colorDurationsMs {
		loopDurationMs += d
	}

	x.flash = &SpriteFlash{
		Colors:           colors,
		ColorDurationsMs: colorDurationsMs,
		TotalDurationMs:  totalDurationMs,
		LoopDurationMs:   loopDurationMs,
	}

	return nil
}

func (x *Sprite) StopFlash() {
	x.flash = nil
}

func (x *Sprite) TickFlash() {
	if x.flash == nil {
		return
	}

	f := x.flash
	f.CounterMs += data.TickMs

	if f.CounterMs >= f.TotalDurationMs {
		x.flash = nil
	}

	colorCounterMs := f.CounterMs % f.LoopDurationMs

	for i, c := range f.ColorDurationsMs {
		if colorCounterMs < c {
			f.ColorIdx = i
			break
		}
		colorCounterMs -= c
	}
}

func (x *Sprite) CurrentColor() (color utils.RelativeColor, ok bool) {
	if x.flash == nil {
		return color, false
	}

	return x.flash.Colors[x.flash.ColorIdx], true
}

func (x *Sprite) CurrentFrame(assetManager *assetmanager.AssetManager) (*ebiten.Image, error) {
	imgAsset, err := assetManager.GetAsset(x.imageAssetTag)
	if err != nil {
		return nil, fmt.Errorf("could not get image asset %s: %v", x.imageAssetTag, err)
	}

	if x.subImageIdx < 0 || x.subImageIdx >= len(imgAsset.Frames) {
		return nil, fmt.Errorf("subImageIdx %d out of bounds for image asset %s", x.subImageIdx, x.imageAssetTag)
	}

	return imgAsset.Frames[x.subImageIdx], nil
}

func (x *Sprite) SetImageAsset(imageAssetTag common.ImageAssetTag) {
	x.imageAssetTag = imageAssetTag
}

func (x *Sprite) SubImageIdx() int {
	return x.subImageIdx
}

func (x *Sprite) SetSubImageIdx(subImageIdx int) {
	x.subImageIdx = subImageIdx
}

func (x *Sprite) LocalOffsetPos() utils.Vec2f {
	return x.offsetPos
}

func (x *Sprite) SetLocalOffsetPos(offset utils.Vec2f) {
	x.offsetPos = offset
}

func (x *Sprite) LocalOffsetRotation() float64 {
	return x.offsetRotation
}

func (x *Sprite) SetLocalOffsetRotation(rotation float64) {
	x.offsetRotation = rotation
}

func (x *Sprite) LocalOffsetScale() float64 {
	return x.offsetScale
}

func (x *Sprite) SetLocalOffsetScale(scale float64) {
	x.offsetScale = scale
}

func (x *Sprite) LocalLayerYOffset(assetManager *assetmanager.AssetManager) (uint16, error) {
	imgAsset, err := assetManager.GetAsset(x.imageAssetTag)
	if err != nil {
		return 0, fmt.Errorf("could not get image asset %s: %v", x.imageAssetTag, err)
	}

	return imgAsset.LayerYOffset, nil
}

func (x *Sprite) Layer() uint8 {
	return x.layer
}

func (x *Sprite) SetLayer(layer uint8) {
	x.layer = layer
}

func (x *Sprite) AllowRotation() bool {
	return x.allowRotation
}

func (x *Sprite) SetAllowRotation(allow bool) {
	x.allowRotation = allow
}

// TODO: Do these once for each entity in a transform system and cache them per tick.
// Unsure of performance benefit, but worth a try.
// Can use a 'dirty' flag and only recalculate entities that have moved
func (x *ECSContainer) SpriteWorldOffsetPos(e common.EntityId) (utils.Vec2f, error) {
	s, ok := x.Sprites.Get(e)
	if !ok {
		return utils.Vec2f{}, fmt.Errorf("entity %d has no sprite", e)
	}

	worldPos, err := x.WorldPos(e)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	worldRot, err := x.WorldRotation(e)
	if err != nil {
		return utils.Vec2f{}, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(worldRot)
	sin := math.Sin(worldRot)

	return utils.Vec2f{
		X: worldPos.X + (s.offsetPos.X*cos - s.offsetPos.Y*sin),
		Y: worldPos.Y + (s.offsetPos.X*sin + s.offsetPos.Y*cos),
	}, nil
}

func (x *ECSContainer) SpriteWorldOffsetRotation(e common.EntityId) (float64, error) {
	s, ok := x.Sprites.Get(e)
	if !ok {
		return 0, fmt.Errorf("entity %d has no sprite", e)
	}

	worldRot, err := x.WorldRotation(e)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	return worldRot + s.offsetRotation, nil
}

func (x *ECSContainer) SpriteWorldOffsetScale(e common.EntityId) (float64, error) {
	s, ok := x.Sprites.Get(e)
	if !ok {
		return 0, fmt.Errorf("entity %d has no sprite", e)
	}

	p := x.Parent(e)
	if p == -1 {
		return s.offsetScale, nil
	}

	pWorldScale, err := x.WorldScale(p)
	if err != nil {
		return 0, fmt.Errorf("error getting world scale of parent entity %d: %v", p, err)
	}

	return pWorldScale * s.offsetScale, nil
}

func (x *ECSContainer) SpriteWorldLayerYOffset(e common.EntityId, assetManager *assetmanager.AssetManager) (uint16, error) {
	s, ok := x.Sprites.Get(e)
	if !ok {
		return 0, fmt.Errorf("entity %d has no sprite", e)
	}

	worldPos, err := x.WorldPos(e)
	if err != nil {
		return 0, fmt.Errorf("error getting world position of entity %d: %v", e, err)
	}

	worldRot, err := x.WorldRotation(e)
	if err != nil {
		return 0, fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	cos := math.Cos(worldRot)
	sin := math.Sin(worldRot)

	imgAsset, err := assetManager.GetAsset(s.imageAssetTag)
	if err != nil {
		return 0, fmt.Errorf("could not get image asset for entity %d: %v", e, err)
	}

	return uint16(worldPos.Y + (float64(imgAsset.LayerYOffset)*sin + float64(imgAsset.LayerYOffset)*cos)), nil
}

func (x Sprite) Copy() Sprite {
	var flashCopy *SpriteFlash
	if x.flash != nil {
		flashCopy = &SpriteFlash{
			Colors:           make([]utils.RelativeColor, len(x.flash.Colors)),
			ColorDurationsMs: make([]int, len(x.flash.ColorDurationsMs)),
			TotalDurationMs:  x.flash.TotalDurationMs,
			ColorIdx:         x.flash.ColorIdx,
			CounterMs:        x.flash.CounterMs,
			LoopDurationMs:   x.flash.LoopDurationMs,
		}
		copy(flashCopy.Colors, x.flash.Colors)
		copy(flashCopy.ColorDurationsMs, x.flash.ColorDurationsMs)
	}

	return Sprite{
		imageAssetTag:  x.imageAssetTag,
		subImageIdx:    x.subImageIdx,
		offsetPos:      x.offsetPos,
		offsetScale:    x.offsetScale,
		offsetRotation: x.offsetRotation,
		layer:          x.layer,
		allowRotation:  x.allowRotation,
		flash:          flashCopy,
	}
}

func (x Sprite) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Sprites.addComponent(e, x.Copy())
}

type spriteDto struct {
	ImageAssetTag  common.ImageAssetTag
	SubImageIdx    int
	OffsetPos      utils.Vec2f
	OffsetScale    float64
	OffsetRotation float64
	Layer          uint8
	AllowRotation  bool
	Flash          *SpriteFlash
}

func (x Sprite) toDto() (ComponentDto, error) {
	x = x.Copy()

	return &spriteDto{
		ImageAssetTag:  x.imageAssetTag,
		SubImageIdx:    x.subImageIdx,
		OffsetPos:      x.offsetPos,
		OffsetScale:    x.offsetScale,
		OffsetRotation: x.offsetRotation,
		Layer:          x.layer,
		AllowRotation:  x.allowRotation,
		Flash:          x.flash,
	}, nil
}

func (x *spriteDto) toComponent() (Component, error) {
	return &Sprite{
		imageAssetTag:  x.ImageAssetTag,
		subImageIdx:    x.SubImageIdx,
		offsetPos:      x.OffsetPos,
		offsetScale:    x.OffsetScale,
		offsetRotation: x.OffsetRotation,
		layer:          x.Layer,
		allowRotation:  x.AllowRotation,
		flash:          x.Flash,
	}, nil
}
