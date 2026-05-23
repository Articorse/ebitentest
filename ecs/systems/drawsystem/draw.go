package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// TODO: This could be optimized further by only ordering the drawing of sprites that overlap, possibly via AABBs?
func DrawFrame(
	screen *ebiten.Image,
	camera utils.Vec2,
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	sprites map[ecscommon.EntityId]*components.Sprite,
	transforms map[ecscommon.EntityId]*components.Transform,
	parents map[ecscommon.EntityId]*components.Parent,
) error {
	sm := components.SpriteManager{}

	batches := make(map[uint8][][]ecscommon.EntityId)
	visitedSprites := make(map[ecscommon.EntityId]struct{})
	i := 0

	for e, _ := range sprites {
		if _, ok := visitedSprites[e]; ok {
			continue
		}
		visitedSprites[e] = struct{}{}

		sprImg, err := sm.GetImage(e, sprites)
		if err != nil {
			return fmt.Errorf("Error getting sprite image for entity %d: %v\n", e, err)
		}

		if sprImg == nil {
			continue
		}

		sprLayer, err := sm.GetLayer(e, sprites)
		if err != nil {
			return fmt.Errorf("Error getting sprite layer for entity %d: %v\n", e, err)
		}

		layer := sprLayer
		batches[layer] = append(batches[layer], []ecscommon.EntityId{})
		batches[layer][i] = append(batches[layer][i], e)

		neighbors, err := getNeighborBatch(e, shg, transforms, sprites, parents)
		if err != nil {
			return err
		}

		for j, n := range neighbors {
			visitedSprites[neighbors[j]] = struct{}{}

			nSprImg, err := sm.GetImage(n, sprites)
			if err != nil {
				return fmt.Errorf("Error getting sprite image for entity %d: %v\n", n, err)
			}

			if nSprImg == nil {
				continue
			}

			batches[layer][i] = append(batches[layer][i], n)
		}

		slices.SortStableFunc(batches[layer][i], func(a, b ecscommon.EntityId) int {
			aTotalY, err := sm.GetWorldLayerYOffset(a, sprites, transforms, parents)
			if err != nil {
				return -1
			}

			bTotalY, err := sm.GetWorldLayerYOffset(b, sprites, transforms, parents)
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

		for _, batchEntity := range batches[layer][i] {
			v, err := sm.GetWorldOffsetPos(batchEntity, sprites, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world offset position of entity %d: %v", batchEntity, err)
			}

			r, err := sm.GetWorldOffsetRotation(batchEntity, sprites, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world offset rotation of entity %d: %v", batchEntity, err)
			}

			s, err := sm.GetWorldOffsetScale(batchEntity, sprites, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world offset scale of entity %d: %v", batchEntity, err)
			}

			img, err := sm.GetImage(batchEntity, sprites)
			if err != nil {
				return fmt.Errorf("Error getting sprite image for entity %d: %v\n", batchEntity, err)
			}

			opts := ebiten.DrawImageOptions{}
			w, h := img.Bounds().Dx(), img.Bounds().Dy()
			opts.GeoM.Scale(s, s)
			opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
			opts.GeoM.Rotate(r)
			opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

			screen.DrawImage(img, &opts)
		}
		i++
	}

	return nil
}

func getNeighborBatch(
	eA ecscommon.EntityId,
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	sprites map[ecscommon.EntityId]*components.Sprite,
	parents map[ecscommon.EntityId]*components.Parent,
) ([]ecscommon.EntityId, error) {
	visitedEntities := make(map[ecscommon.EntityId]struct{})
	neighbors, _, err := getNeighborsRecursive(eA, shg, visitedEntities, transforms, sprites, parents)
	if err != nil {
		return nil, err
	}

	return neighbors, nil
}

func getNeighborsRecursive(
	eA ecscommon.EntityId,
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	visitedEntities map[ecscommon.EntityId]struct{},
	transforms map[ecscommon.EntityId]*components.Transform,
	sprites map[ecscommon.EntityId]*components.Sprite,
	parents map[ecscommon.EntityId]*components.Parent,
) (neighbors []ecscommon.EntityId, _visited map[ecscommon.EntityId]struct{}, err error) {
	tm := components.TransformManager{}
	sm := components.SpriteManager{}

	visitedEntities[eA] = struct{}{}

	aWorldPos, err := tm.GetWorldPos(eA, transforms, parents)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting world position of entity %d: %v", eA, err)
	}

	startCellX := int(aWorldPos.X / data.SpatialHashGridCellSize)
	startCellY := int(aWorldPos.Y / data.SpatialHashGridCellSize)

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for _, eB := range shg[ecscommon.CellKey{X: startCellX + dx, Y: startCellY + dy}] {
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

				aLayer, err := sm.GetLayer(eA, sprites)
				if err != nil {
					return nil, nil, fmt.Errorf("error getting sprite layer for entity %d: %v", eA, err)
				}

				bLayer, err := sm.GetLayer(eB, sprites)
				if err != nil {
					return nil, nil, fmt.Errorf("error getting sprite layer for entity %d: %v", eB, err)
				}

				if aLayer != bLayer {
					continue
				}

				n, v, err := getNeighborsRecursive(eB, shg, visitedEntities, transforms, sprites, parents)
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
