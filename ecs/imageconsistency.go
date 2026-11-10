package ecs

import (
	"ebittest/assetmanager"
	"ebittest/ecs/common"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Implemented by ability params that draw images when they're used, such as a slash's, so that the consistency check
// can find images that would only fail once the ability is used
type ImageUser interface {
	// The frames of each image that using the ability shows
	UsedImages() map[common.ImageAssetTag][]int
}

// The images the consistency check looks for. Without them, it doesn't check images.
func (x *ECSContainer) SetAssets(assets *assetmanager.AssetManager) {
	x.assets = assets
}

// Every image an entity draws, or would draw through what it spawns or the abilities it uses, is in the asset list,
// with the frames it shows
func (x *ECSContainer) checkImages() error {
	if x.assets == nil {
		return nil
	}

	var errs error
	check := func(e common.EntityId, comp Component) {
		errs = errors.Join(errs, x.checkComponentImages(fmt.Sprintf("entity %d", e), comp))
	}
	for e, c := range Query1(&x.Sprites) {
		check(e, c)
	}
	for e, c := range Query1(&x.Animations) {
		check(e, c)
	}
	for e, c := range Query1(&x.Abilities) {
		check(e, c)
	}
	for e, c := range Query1(&x.Equipments) {
		check(e, c)
	}
	for e, c := range Query1(&x.Deathrattles) {
		check(e, c)
	}
	for e, c := range Query1(&x.Spawners) {
		check(e, c)
	}
	return errs
}

func (x *ECSContainer) checkComponentImages(owner string, comp Component) error {
	switch c := comp.(type) {
	case *Sprite:
		return x.checkImage(owner+"'s sprite", c.imageAssetTag, []int{c.subImageIdx})
	case *Animation:
		var errs error
		for _, frames := range c.stateFrames {
			shown := make([]int, len(frames))
			for i, f := range frames {
				shown[i] = int(f.FrameIdx)
			}
			errs = errors.Join(errs, x.checkImage(owner+"'s animation", c.sheetAssetTag, shown))
		}
		return errs
	case *Abilities:
		return x.checkAbilityImages(owner, c.abilities[:])
	case *Equipment:
		errs := x.checkAbilityImages(owner+"'s equipment", c.abilities[:])
		if c.worn != "" {
			errs = errors.Join(errs, x.checkImage(owner+"'s equipment when worn", c.worn, nil))
		}
		return errs
	case *Deathrattle:
		return x.checkAbilityImages(owner+"'s deathrattle", []EntityAbility{c.ability})
	case *Spawner:
		var errs error
		for _, spawned := range c.components {
			errs = errors.Join(errs, x.checkComponentImages(owner+"'s spawn", spawned))
		}
		return errs
	}
	return nil
}

func (x *ECSContainer) checkAbilityImages(owner string, abilities []EntityAbility) error {
	var errs error
	for _, a := range abilities {
		user, ok := a.Params.(ImageUser)
		if !ok {
			continue
		}
		images := user.UsedImages()
		for _, tag := range slices.Sorted(maps.Keys(images)) {
			errs = errors.Join(errs, x.checkImage(fmt.Sprintf("%s's ability %v", owner, a.Def.AbilityId), tag, images[tag]))
		}
	}
	return errs
}

func (x *ECSContainer) checkImage(what string, tag common.ImageAssetTag, frames []int) error {
	asset, err := x.assets.GetAsset(tag)
	if err != nil {
		return fmt.Errorf("%s draws %s, which isn't in the asset list", what, tag)
	}
	for _, f := range frames {
		if f < 0 || f >= len(asset.Frames) {
			return fmt.Errorf("%s shows frame %d of %s, which has %d", what, f, tag, len(asset.Frames))
		}
	}
	return nil
}
