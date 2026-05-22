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
	tm := components.TransformManager{}
	batches := make(map[uint8][][]ecscommon.EntityId)
	visitedSprites := make(map[ecscommon.EntityId]struct{})
	i := 0

	for e, sprComp := range sprites {
		if _, ok := visitedSprites[e]; ok {
			continue
		}
		visitedSprites[e] = struct{}{}

		_, ok := transforms[e]
		if !ok {
			return &ecscommon.ErrorMissingComponentDependency{
				Entity:           e,
				PresentComponent: "Sprite",
				MissingComponent: "Transform",
			}
		}

		if sprComp.Image == nil {
			continue
		}

		layer := sprComp.Layer
		batches[layer] = append(batches[layer], []ecscommon.EntityId{})
		batches[layer][i] = append(batches[layer][i], e)

		neighbors, err := getNeighborBatch(e, shg, transforms, sprites, parents)
		if err != nil {
			return err
		}

		for j, n := range neighbors {
			sprCompN, ok := sprites[n]
			if !ok {
				return &ecscommon.ErrorMissingExpectedComponent{
					Entity:           n,
					MissingComponent: "Sprite",
				}
			}

			visitedSprites[neighbors[j]] = struct{}{}

			if sprCompN.Image == nil {
				continue
			}

			batches[layer][i] = append(batches[layer][i], n)
		}

		slices.SortStableFunc(batches[layer][i], func(a, b ecscommon.EntityId) int {
			sprCompA, ok := sprites[a]
			if !ok {
				err = &ecscommon.ErrorMissingExpectedComponent{
					Entity:           a,
					MissingComponent: "Sprite",
				}
				return -1
			}
			sprCompB, ok := sprites[b]
			if !ok {
				err = &ecscommon.ErrorMissingExpectedComponent{
					Entity:           b,
					MissingComponent: "Sprite",
				}
				return -1
			}

			worldPosA, err := tm.GetWorldPos(a, transforms, parents)
			if err != nil {
				return -1
			}

			worldPosB, err := tm.GetWorldPos(b, transforms, parents)
			if err != nil {
				return -1
			}

			aTotalY := uint64(worldPosA.Y) + uint64(sprCompA.LayerYOffset)
			bTotalY := uint64(worldPosB.Y) + uint64(sprCompB.LayerYOffset)

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
			sprCompBE, _ := sprites[batchEntity]

			beWorldPos, err := tm.GetWorldPos(batchEntity, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world position of entity %d: %v", batchEntity, err)
			}

			beWorldRot, err := tm.GetWorldRotation(batchEntity, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world rotation of entity %d: %v", batchEntity, err)
			}

			beWorldScale, err := tm.GetWorldScale(batchEntity, transforms, parents)
			if err != nil {
				return fmt.Errorf("error getting world scale of entity %d: %v", batchEntity, err)
			}

			v := beWorldPos.Add(sprCompBE.OffsetPos)
			r := beWorldRot + sprCompBE.OffsetRotation
			s := beWorldScale * sprCompBE.OffsetScale

			opts := ebiten.DrawImageOptions{}
			w, h := sprCompBE.Image.Bounds().Dx(), sprCompBE.Image.Bounds().Dy()
			opts.GeoM.Scale(s, s)
			opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
			opts.GeoM.Rotate(r)
			opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

			screen.DrawImage(sprCompBE.Image, &opts)
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

	sprCompA, ok := sprites[eA]
	if !ok {
		return nil, nil, &ecscommon.ErrorMissingExpectedComponent{
			Entity:           eA,
			MissingComponent: "Sprite",
		}
	}

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

				sprCompB, ok := sprites[eB]
				if !ok {
					continue
				}

				_, ok = transforms[eB]
				if !ok {
					return nil, nil, &ecscommon.ErrorMissingComponentDependency{
						Entity:           eB,
						PresentComponent: "Sprite",
						MissingComponent: "Transform",
					}
				}

				if sprCompA.Layer != sprCompB.Layer {
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
