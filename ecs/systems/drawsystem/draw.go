package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TODO: This could be optimized further by only ordering the drawing of sprites that overlap, possibly via AABBs?
func DrawSprites(
	screen *ebiten.Image,
	camera utils.Vec2,
	shg map[common.CellKey][]common.EntityId,
	ecs *ecs.ECS,
) error {
	sm := ecs.SpriteManager
	pm := ecs.ParentManager
	tm := ecs.TransformManager

	batches := make(map[uint8][][]common.EntityId)
	visitedSprites := make(map[common.EntityId]struct{})
	layerIdxMap := make(map[uint8]uint64)
	drawWindow := [2]utils.Vec2{
		utils.Vec2{X: camera.X - data.SpatialHashGridCellSize, Y: camera.Y - data.SpatialHashGridCellSize},
		utils.Vec2{X: camera.X + data.CameraWidth + data.SpatialHashGridCellSize, Y: camera.Y + data.CameraHeight + data.SpatialHashGridCellSize},
	}

	for _, e := range ecs.Sprites.GetEntities() {
		eWorldPos, err := tm.GetWorldPos(e, ecs)
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

		sprImg, err := sm.GetImage(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting sprite image for entity %d: %v", e, err)
		}

		if sprImg == nil {
			continue
		}

		layer, err := sm.GetLayer(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting sprite layer for entity %d: %v", e, err)
		}

		i, ok := layerIdxMap[layer]
		if !ok {
			layerIdxMap[layer] = 0
			i = 0
		} else {
			layerIdxMap[layer] = layerIdxMap[layer] + 1
			i = layerIdxMap[layer]
		}

		batches[layer] = append(batches[layer], []common.EntityId{})
		batches[layer][i] = append(batches[layer][i], e)

		neighbors, err := getNeighborBatch(e, shg, ecs)
		if err != nil {
			return err
		}

		for j, n := range neighbors {
			visitedSprites[neighbors[j]] = struct{}{}

			nSprImg, err := sm.GetImage(n, ecs)
			if err != nil {
				return fmt.Errorf("error getting sprite image for entity %d: %v", n, err)
			}

			if nSprImg == nil {
				continue
			}

			batches[layer][i] = append(batches[layer][i], n)
		}

		if len(batches[layer][i]) > 1 {
			var err error

			hierarchies, err := pm.GetOrderedHierarchies(batches[layer][i], ecs)
			if err != nil {
				return fmt.Errorf("error getting ordered hierarchies for batch in layer %d: %v", layer, err)
			}

			slices.SortStableFunc(hierarchies, func(aRoot, bRoot [][]common.EntityId) int {
				a := aRoot[0][0]
				b := bRoot[0][0]

				aTotalY, err := sm.GetWorldLayerYOffset(a, ecs)
				if err != nil {
					return -1
				}

				bTotalY, err := sm.GetWorldLayerYOffset(b, ecs)
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

		i++
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
				v, err := sm.GetWorldOffsetPos(batchEntity, ecs)
				if err != nil {
					return fmt.Errorf("error getting ecs offset position of batchEntity %d: %v", batchEntity, err)
				}

				r := 0.0
				allowRot, err := sm.GetAllowRotation(batchEntity, ecs)
				if err != nil {
					return fmt.Errorf("error getting allow rotation of batchEntity %d: %v", batchEntity, err)
				}
				if allowRot {
					r, err = sm.GetWorldOffsetRotation(batchEntity, ecs)
					if err != nil {
						return fmt.Errorf("error getting ecs offset rotation of batchEntity %d: %v", batchEntity, err)
					}
				}

				s, err := sm.GetWorldOffsetScale(batchEntity, ecs)
				if err != nil {
					return fmt.Errorf("error getting ecs offset scale of batchEntity %d: %v", batchEntity, err)
				}

				img, err := sm.GetImage(batchEntity, ecs)
				if err != nil {
					return fmt.Errorf("error getting sprite image for batchEntity %d: %v", batchEntity, err)
				}

				opts := ebiten.DrawImageOptions{}
				w, h := img.Bounds().Dx(), img.Bounds().Dy()
				opts.GeoM.Scale(s, s)
				opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
				opts.GeoM.Rotate(r)
				opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

				color, ok, err := sm.GetCurrentColor(batchEntity, ecs)
				if err != nil {
					return fmt.Errorf("error getting current color of batchEntity %d: %v", batchEntity, err)
				}
				if ok {
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

func DrawFloatingTexts(screen *ebiten.Image, ecs *ecs.ECS) error {
	tm := ecs.TransformManager
	ftm := ecs.FloatingTextManager

	for _, e := range ecs.FloatingTexts.GetEntities() {
		ecsPos, err := tm.GetWorldPos(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting ecs position of floating text entity %d: %v", e, err)
		}

		offset, err := ftm.GetOffset(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting offset of floating text entity %d: %v", e, err)
		}

		textValue, err := ftm.GetText(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting text of floating text entity %d: %v", e, err)
		}

		color, err := ftm.GetColor(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting color of floating text entity %d: %v", e, err)
		}

		face, err := ftm.GetFace(e, ecs)
		if err != nil {
			return fmt.Errorf("error getting font face of floating text entity %d: %v", e, err)
		}

		op := &text.DrawOptions{}
		tX := ecsPos.X + offset.X - ecs.Camera.X
		tY := ecsPos.Y + offset.Y - ecs.Camera.Y
		w, h := text.Measure(textValue, &face, 0)
		op.GeoM.Translate(tX-w/2, tY-h/2)
		op.ColorScale.ScaleWithColor(color)

		text.Draw(screen, textValue, &face, op)
	}

	return nil
}

func getNeighborBatch(
	eA common.EntityId,
	shg map[common.CellKey][]common.EntityId,
	ecs *ecs.ECS,
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
	shg map[common.CellKey][]common.EntityId,
	visitedEntities map[common.EntityId]struct{},
	ecs *ecs.ECS,
) (neighbors []common.EntityId, _visited map[common.EntityId]struct{}, err error) {
	tm := ecs.TransformManager
	sm := ecs.SpriteManager

	if !ecs.Sprites.HasComponent(eA) {
		return nil, nil, fmt.Errorf("entity %d does not have a sprite component", eA)
	}

	visitedEntities[eA] = struct{}{}

	aWorldPos, err := tm.GetWorldPos(eA, ecs)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting ecs position of entity %d: %v", eA, err)
	}

	aLayer, err := sm.GetLayer(eA, ecs)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting sprite layer for entity %d: %v", eA, err)
	}

	startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
	startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for _, eB := range shg[common.CellKey{X: startCellX + dx, Y: startCellY + dy}] {
				if !ecs.Sprites.HasComponent(eB) {
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

				bLayer, err := sm.GetLayer(eB, ecs)
				if err != nil {
					return nil, nil, fmt.Errorf("error getting sprite layer for entity %d: %v", eB, err)
				}

				if aLayer != bLayer {
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
