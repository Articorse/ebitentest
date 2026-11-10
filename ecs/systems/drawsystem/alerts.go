package drawsystem

import (
	"ebittest/assetmanager"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Where an exclamation mark goes over each entity in the window that's alerted: its top-left corner, a pixel above the
// top of the entity's sprite, centred on it
func AlertMarks(
	window [2]utils.Vec2f,
	alerted func(common.EntityId) bool,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) ([]utils.Vec2f, error) {
	asset, err := assetManager.GetAsset(common.AssetImageExclamation)
	if err != nil {
		return nil, fmt.Errorf("error getting the exclamation mark: %v", err)
	}
	mark := asset.Frames[0].Bounds().Size()

	marks := []utils.Vec2f{}
	for e, spr := range ecs.Query1(&ecsContainer.Sprites, ecs.Without(&ecsContainer.Disabled)) {
		if !alerted(e) {
			continue
		}
		pos, err := ecsContainer.SpriteWorldOffsetPos(e)
		if err != nil {
			return nil, fmt.Errorf("error getting the sprite position of entity %d: %v", e, err)
		}
		if pos.X < window[0].X || pos.X > window[1].X || pos.Y < window[0].Y || pos.Y > window[1].Y {
			continue
		}
		img, err := spr.CurrentFrame(assetManager)
		if err != nil {
			return nil, fmt.Errorf("error getting sprite image for entity %d: %v", e, err)
		}
		if img == nil {
			continue
		}
		scale, err := ecsContainer.SpriteWorldOffsetScale(e)
		if err != nil {
			return nil, fmt.Errorf("error getting the sprite scale of entity %d: %v", e, err)
		}
		top := pos.Y - float64(img.Bounds().Dy())*scale/2
		marks = append(marks, utils.Vec2f{X: pos.X - float64(mark.X)/2, Y: top - 1 - float64(mark.Y)})
	}
	return marks, nil
}

// Over every sprite, so nothing standing in front hides it, but drawn into the world before the light and fog, which
// darken and hide it like the entity under it. The red mark, for what's getting worse, is the same size and goes instead
// of the other.
func DrawAlerts(
	dst *ebiten.Image,
	camera utils.Vec2f,
	alerted, worsening func(common.EntityId) bool,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	for _, kind := range []struct {
		tag   common.ImageAssetTag
		marks func(common.EntityId) bool
	}{
		{common.AssetImageExclamation, func(e common.EntityId) bool { return alerted(e) && !worsening(e) }},
		{common.AssetImageExclamationRed, worsening},
	} {
		marks, err := AlertMarks(DrawWindow(camera, ImageSize(dst)), kind.marks, ecsContainer, assetManager)
		if err != nil {
			return err
		}
		asset, err := assetManager.GetAsset(kind.tag)
		if err != nil {
			return fmt.Errorf("error getting the exclamation mark: %v", err)
		}
		for _, m := range marks {
			opts := ebiten.DrawImageOptions{}
			opts.GeoM.Translate(m.X-camera.X, m.Y-camera.Y)
			dst.DrawImage(asset.Frames[0], &opts)
		}
	}
	return nil
}
