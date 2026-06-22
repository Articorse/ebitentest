package ecs

import (
	"bytes"
	"ebittest/ecs/common"
	"ebittest/utils"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type floatingTextManager struct{}

func NewFloatingTextComponent(ptext string, pos utils.Vec2, size float64, color color.RGBA) *floatingText {
	fontBytes, _ := os.ReadFile("/usr/share/fonts/noto/NotoSansMono-Black.ttf")

	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontBytes))
	if err != nil {
		log.Fatal(err)
	}

	face := text.GoTextFace{
		Source: source,
		Size:   size,
	}

	return &floatingText{
		text:   ptext,
		offset: pos,
		size:   size,
		color:  color,
		face:   face,
	}
}

func (floatingTextManager) GetText(e common.EntityId, ecsContainer *ECSContainer) (string, error) {
	textComp, err := ecsContainer.FloatingTexts.getComponent(e)
	if err != nil {
		return "", err
	}

	return textComp.text, nil
}

func (floatingTextManager) GetOffset(e common.EntityId, ecsContainer *ECSContainer) (utils.Vec2, error) {
	textComp, err := ecsContainer.FloatingTexts.getComponent(e)
	if err != nil {
		return utils.Vec2{}, err
	}

	return textComp.offset, nil
}

func (floatingTextManager) GetSize(e common.EntityId, ecsContainer *ECSContainer) (float64, error) {
	textComp, err := ecsContainer.FloatingTexts.getComponent(e)
	if err != nil {
		return 0, err
	}

	return textComp.size, nil
}

func (floatingTextManager) GetColor(e common.EntityId, ecsContainer *ECSContainer) (color.RGBA, error) {
	textComp, err := ecsContainer.FloatingTexts.getComponent(e)
	if err != nil {
		return color.RGBA{R: 255, G: 0, B: 255, A: 255}, err
	}

	return textComp.color, nil
}

func (floatingTextManager) GetFace(e common.EntityId, ecsContainer *ECSContainer) (text.GoTextFace, error) {
	textComp, err := ecsContainer.FloatingTexts.getComponent(e)
	if err != nil {
		return text.GoTextFace{}, err
	}

	return textComp.face, nil
}
