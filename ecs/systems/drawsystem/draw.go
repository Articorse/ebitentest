package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs/components"
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
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
) error {
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

		neighbors, err := getNeighborBatch(shg, transforms, sprites, e)
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

			traCompA, ok := transforms[a]
			if !ok {
				err = &ecscommon.ErrorMissingComponentDependency{
					Entity:           a,
					PresentComponent: "Sprite",
					MissingComponent: "Transform",
				}
				return -1
			}
			traCompB, ok := transforms[b]
			if !ok {
				err = &ecscommon.ErrorMissingComponentDependency{
					Entity:           b,
					PresentComponent: "Sprite",
					MissingComponent: "Transform",
				}
				return -1
			}

			aTotalY := uint64(traCompA.GetPos().Y) + uint64(sprCompA.LayerYOffset)
			bTotalY := uint64(traCompB.GetPos().Y) + uint64(sprCompB.LayerYOffset)

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
			traCompBE, _ := transforms[batchEntity]

			v := traCompBE.GetPos().Add(sprCompBE.OffsetPos)
			r := traCompBE.GetRotation() + sprCompBE.OffsetRotation
			s := traCompBE.GetScale() * sprCompBE.OffsetScale

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
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	sprites map[ecscommon.EntityId]*components.Sprite,
	eA ecscommon.EntityId,
) ([]ecscommon.EntityId, error) {
	visitedEntities := make(map[ecscommon.EntityId]struct{})
	neighbors, _, err := getNeighborsRecursive(shg, transforms, sprites, eA, visitedEntities)
	if err != nil {
		return nil, err
	}

	return neighbors, nil
}

func getNeighborsRecursive(
	shg map[ecscommon.CellKey][]ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*components.Transform,
	sprites map[ecscommon.EntityId]*components.Sprite,
	eA ecscommon.EntityId,
	visitedEntities map[ecscommon.EntityId]struct{},
) (neighbors []ecscommon.EntityId, _visited map[ecscommon.EntityId]struct{}, err error) {
	traCompA, ok := transforms[eA]
	if !ok {
		return nil, nil, &ecscommon.ErrorMissingComponentDependency{
			Entity:           eA,
			PresentComponent: "Sprite",
			MissingComponent: "Transform",
		}
	}

	sprCompA, ok := sprites[eA]
	if !ok {
		return nil, nil, &ecscommon.ErrorMissingExpectedComponent{
			Entity:           eA,
			MissingComponent: "Sprite",
		}
	}

	visitedEntities[eA] = struct{}{}

	startCellX := int(traCompA.GetPos().X / data.SpatialHashGridCellSize)
	startCellY := int(traCompA.GetPos().Y / data.SpatialHashGridCellSize)

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

				n, v, err := getNeighborsRecursive(shg, transforms, sprites, eB, visitedEntities)
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
