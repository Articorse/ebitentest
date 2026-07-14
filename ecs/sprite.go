package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

type SpriteFlash struct {
	Colors           []utils.RelativeColor
	ColorDurationsMs []int
	TotalDurationMs  int

	ColorIdx       int
	CounterMs      int
	LoopDurationMs int
}

type sprite struct {
	imageAssetTag  common.ImageAssetTag
	subImageIdx    int
	offsetPos      utils.Vec2f
	offsetScale    float64
	offsetRotation float64
	layer          uint8
	allowRotation  bool
	flash          *SpriteFlash
}

func (sprite) isComponent() {}

func (x sprite) Copy() sprite {
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

	return sprite{
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

func (spriteDto) isComponentDto() {}

func (x sprite) ToDto() spriteDto {
	return spriteDto{
		ImageAssetTag:  x.imageAssetTag,
		SubImageIdx:    x.subImageIdx,
		OffsetPos:      x.offsetPos,
		OffsetScale:    x.offsetScale,
		OffsetRotation: x.offsetRotation,
		Layer:          x.layer,
		AllowRotation:  x.allowRotation,
		Flash:          x.flash,
	}
}

func (x *spriteDto) ToComponent() *sprite {
	return &sprite{
		imageAssetTag:  x.ImageAssetTag,
		subImageIdx:    x.SubImageIdx,
		offsetPos:      x.OffsetPos,
		offsetScale:    x.OffsetScale,
		offsetRotation: x.OffsetRotation,
		layer:          x.Layer,
		allowRotation:  x.AllowRotation,
		flash:          x.Flash,
	}
}
