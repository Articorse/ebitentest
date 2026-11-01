package drawsystem

import (
	"bytes"
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TODO: This could be optimized further by only ordering the drawing of sprites that overlap, possibly via AABBs?
func DrawSprites(
	screen *ebiten.Image,
	camera utils.Vec2f,
	shg map[utils.Vec2i][]common.EntityId,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	batches := make(map[uint8][][]common.EntityId)
	visitedSprites := make(map[common.EntityId]struct{})
	layerIdxMap := make(map[uint8]uint64)
	drawWindow := [2]utils.Vec2f{
		utils.Vec2f{X: camera.X - data.SpatialHashGridCellSize, Y: camera.Y - data.SpatialHashGridCellSize},
		utils.Vec2f{X: camera.X + data.CameraWidth + data.SpatialHashGridCellSize, Y: camera.Y + data.CameraHeight + data.SpatialHashGridCellSize},
	}

	for e, spr := range ecs.Query1(&ecsContainer.Sprites) {
		eWorldPos, err := ecsContainer.WorldPos(e)
		if err != nil {
			return fmt.Errorf("error getting ecs position of entity %d: %v", e, err)
		}

		if eWorldPos.X < drawWindow[0].X ||
			eWorldPos.X > drawWindow[1].X ||
			eWorldPos.Y < drawWindow[0].Y ||
			eWorldPos.Y > drawWindow[1].Y {
			continue
		}

		if _, ok := visitedSprites[e]; ok {
			continue
		}
		visitedSprites[e] = struct{}{}

		sprImg, err := spr.CurrentFrame(assetManager)
		if err != nil {
			return fmt.Errorf("error getting sprite image for entity %d: %v", e, err)
		}

		if sprImg == nil {
			continue
		}

		layer := spr.Layer()

		var i uint64
		_, ok := layerIdxMap[layer]
		if !ok {
			layerIdxMap[layer] = 0
			i = 0
		} else {
			layerIdxMap[layer] = layerIdxMap[layer] + 1
			i = layerIdxMap[layer]
		}

		batches[layer] = append(batches[layer], []common.EntityId{})
		batches[layer][i] = append(batches[layer][i], e)

		neighbors, err := getNeighborBatch(e, shg, ecsContainer)
		if err != nil {
			return err
		}

		for j, n := range neighbors {
			visitedSprites[neighbors[j]] = struct{}{}

			nSpr, ok := ecsContainer.Sprites.Get(n)
			if !ok {
				continue
			}

			nSprImg, err := nSpr.CurrentFrame(assetManager)
			if err != nil {
				return fmt.Errorf("error getting sprite image for neighbor entity %d: %v", n, err)
			}

			if nSprImg == nil {
				continue
			}

			batches[layer][i] = append(batches[layer][i], n)
		}

		if len(batches[layer][i]) > 1 {
			var err error

			hierarchies, err := ecsContainer.OrderedHierarchies(batches[layer][i])
			if err != nil {
				return fmt.Errorf("error getting ordered hierarchies for batch in layer %d: %v", layer, err)
			}

			slices.SortStableFunc(hierarchies, func(aRoot, bRoot [][]common.EntityId) int {
				a := aRoot[0][0]
				b := bRoot[0][0]

				aTotalY, err := ecsContainer.SpriteWorldLayerYOffset(a, assetManager)
				if err != nil {
					return -1
				}

				bTotalY, err := ecsContainer.SpriteWorldLayerYOffset(b, assetManager)
				if err != nil {
					return -1
				}

				if aTotalY == bTotalY {
					return int(b - a)
				}

				lower := aTotalY > bTotalY
				if lower {
					return 1
				}
				return -1
			})
			if err != nil {
				return err
			}

			flatOrder := []common.EntityId{}
			for _, hBatch := range hierarchies {
				for _, hLevel := range hBatch {
					slices.SortStableFunc(hLevel, func(a common.EntityId, b common.EntityId) int { return int(b - a) })
					flatOrder = append(flatOrder, hLevel...)
				}
			}

			batches[layer][i] = flatOrder
		}
	}

	batchKeys := maps.Keys(batches)
	var orderedKeys []uint8
	for layerNum := range batchKeys {
		orderedKeys = append(orderedKeys, layerNum)
	}

	slices.Sort(orderedKeys)

	for _, layer := range orderedKeys {
		for _, batch := range batches[layer] {
			for _, batchEntity := range batch {
				spr, ok := ecsContainer.Sprites.Get(batchEntity)
				if !ok {
					return fmt.Errorf("batchEntity %d has no sprite", batchEntity)
				}

				v, err := ecsContainer.SpriteWorldOffsetPos(batchEntity)
				if err != nil {
					return fmt.Errorf("error getting ecs offset position of batchEntity %d: %v", batchEntity, err)
				}

				r := 0.0
				if spr.AllowRotation() {
					r, err = ecsContainer.SpriteWorldOffsetRotation(batchEntity)
					if err != nil {
						return fmt.Errorf("error getting ecs offset rotation of batchEntity %d: %v", batchEntity, err)
					}
				}

				s, err := ecsContainer.SpriteWorldOffsetScale(batchEntity)
				if err != nil {
					return fmt.Errorf("error getting ecs offset scale of batchEntity %d: %v", batchEntity, err)
				}

				img, err := spr.CurrentFrame(assetManager)
				if err != nil {
					return fmt.Errorf("error getting sprite image for batchEntity %d: %v", batchEntity, err)
				}

				opts := ebiten.DrawImageOptions{}
				w, h := img.Bounds().Dx(), img.Bounds().Dy()
				opts.GeoM.Scale(s, s)
				opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
				opts.GeoM.Rotate(r)
				opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

				if color, ok := spr.CurrentColor(); ok {
					opts.ColorScale.Scale(
						color.R,
						color.G,
						color.B,
						color.A,
					)
				}

				screen.DrawImage(img, &opts)
			}
		}
	}

	return nil
}

func DrawFloatingTexts(screen *ebiten.Image, ecs *ecs.ECSContainer) error {
	for _, e := range ecs.FloatingTexts.GetEntities() {
		ecsPos, err := ecs.WorldPos(e)
		if err != nil {
			return fmt.Errorf("error getting ecs position of floating text entity %d: %v", e, err)
		}

		ft, _ := ecs.FloatingTexts.Get(e)
		offset, textValue, color, size := ft.Offset(), ft.Text(), ft.Color(), ft.Size()

		face, err := floatingTextFace(size)
		if err != nil {
			return fmt.Errorf("error getting font for floating text entity %d: %v", e, err)
		}

		op := &text.DrawOptions{}
		tX := ecsPos.X + offset.X - ecs.Camera.X
		tY := ecsPos.Y + offset.Y - ecs.Camera.Y
		w, h := text.Measure(textValue, face, 0)
		op.GeoM.Translate(tX-w/2, tY-h/2)
		op.ColorScale.ScaleWithColor(color)

		text.Draw(screen, textValue, face, op)
	}

	return nil
}

const floatingTextFontPath = "/usr/share/fonts/noto/NotoSansMono-Black.ttf"

// Loaded once: parsing the font is slow, and drawn glyphs are only reused within the same source
var floatingTextFont struct {
	loaded bool
	source *text.GoTextFaceSource
	err    error
	faces  map[float64]*text.GoTextFace
}

func floatingTextFace(size float64) (*text.GoTextFace, error) {
	f := &floatingTextFont
	if !f.loaded {
		f.loaded = true
		f.faces = make(map[float64]*text.GoTextFace)

		fontBytes, err := os.ReadFile(floatingTextFontPath)
		if err != nil {
			f.err = fmt.Errorf("error reading font: %w", err)
		} else if f.source, err = text.NewGoTextFaceSource(bytes.NewReader(fontBytes)); err != nil {
			f.err = fmt.Errorf("error parsing font: %w", err)
		}
	}
	if f.err != nil {
		return nil, f.err
	}

	face, ok := f.faces[size]
	if !ok {
		face = &text.GoTextFace{Source: f.source, Size: size}
		f.faces[size] = face
	}
	return face, nil
}

func getNeighborBatch(
	eA common.EntityId,
	shg map[utils.Vec2i][]common.EntityId,
	ecs *ecs.ECSContainer,
) ([]common.EntityId, error) {
	if !ecs.Sprites.HasComponent(eA) {
		return nil, fmt.Errorf("entity %d does not have a sprite component", eA)
	}

	visitedEntities := make(map[common.EntityId]struct{})
	neighbors, _, err := getNeighborsRecursive(eA, shg, visitedEntities, ecs)
	if err != nil {
		return nil, err
	}

	return neighbors, nil
}

func getNeighborsRecursive(
	eA common.EntityId,
	shg map[utils.Vec2i][]common.EntityId,
	visitedEntities map[common.EntityId]struct{},
	ecs *ecs.ECSContainer,
) (neighbors []common.EntityId, _visited map[common.EntityId]struct{}, err error) {
	aSpr, ok := ecs.Sprites.Get(eA)
	if !ok {
		return nil, nil, fmt.Errorf("entity %d does not have a sprite component", eA)
	}

	visitedEntities[eA] = struct{}{}

	aWorldPos, err := ecs.WorldPos(eA)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting ecs position of entity %d: %v", eA, err)
	}

	aLayer := aSpr.Layer()

	startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
	startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for _, eB := range shg[utils.Vec2i{X: startCellX + dx, Y: startCellY + dy}] {
				bSpr, ok := ecs.Sprites.Get(eB)
				if !ok {
					return nil, nil, nil
				}

				if eA == eB {
					continue
				}

				if slices.Contains(neighbors, eB) {
					continue
				}

				if _, ok := visitedEntities[eB]; ok {
					continue
				}
				visitedEntities[eB] = struct{}{}

				if aLayer != bSpr.Layer() {
					continue
				}

				n, v, err := getNeighborsRecursive(eB, shg, visitedEntities, ecs)
				if err != nil {
					return nil, nil, err
				}

				maps.Copy(visitedEntities, v)
				neighbors = append(neighbors, eB)
				neighbors = slices.Concat(neighbors, n)
			}
		}
	}

	return neighbors, visitedEntities, nil
}
