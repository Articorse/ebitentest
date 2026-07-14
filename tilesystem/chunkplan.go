package tilesystem

import (
	"ebittest/ecs"
	"ebittest/utils"
	"errors"
	"fmt"
	"sort"
)

func (cc *ChunkContainer) ComputeChunkSets(
	ecsCont *ecs.ECSContainer,
) (
	required map[utils.Vec2i]struct{},
	toBeAdded []utils.Vec2i,
	toBeRemoved []utils.Vec2i,
	priority map[utils.Vec2i]int,
	err error,
) {
	required, priority, berr := buildRequiredAndPriority(ecsCont)
	if berr != nil {
		err = fmt.Errorf("failed to build required and priority sets: %w", berr)
	}

	for pos := range required {
		if _, ok := cc.chunks[pos]; !ok {
			toBeAdded = append(toBeAdded, pos)
		}
	}

	for pos := range cc.chunks {
		if _, ok := required[pos]; !ok {
			toBeRemoved = append(toBeRemoved, pos)
		}
	}

	sort.Slice(toBeAdded, func(i, j int) bool {
		pi, pj := priority[toBeAdded[i]], priority[toBeAdded[j]]
		if pi != pj {
			return pi < pj
		}
		if toBeAdded[i].X != toBeAdded[j].X {
			return toBeAdded[i].X < toBeAdded[j].X
		}
		return toBeAdded[i].Y < toBeAdded[j].Y
	})

	return required, toBeAdded, toBeRemoved, priority, err
}

func buildRequiredAndPriority(ecsCont *ecs.ECSContainer) (
	required map[utils.Vec2i]struct{},
	priority map[utils.Vec2i]int,
	err error,
) {
	tm := ecsCont.TransformManager
	clm := ecsCont.ChunkLoaderManager

	required = make(map[utils.Vec2i]struct{})
	priority = make(map[utils.Vec2i]int)
	loaders := ecsCont.ChunkLoaders.GetEntities()

	for _, eId := range loaders {
		worldPos, ierr := tm.GetWorldPos(eId, ecsCont)
		if ierr != nil {
			err = errors.Join(err, fmt.Errorf("error getting world position of entity %d: %v", eId, ierr))
			continue
		}

		center := utils.WorldPosToChunkGridPos(worldPos)

		radius, ierr := clm.GetRadius(eId, ecsCont)
		if ierr != nil {
			err = errors.Join(err, fmt.Errorf("error getting radius of entity %d: %v", eId, ierr))
			continue
		}

		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				pos := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
				required[pos] = struct{}{}

				d := dx*dx + dy*dy
				if old, ok := priority[pos]; !ok || d < old {
					priority[pos] = d
				}
			}
		}
	}

	return required, priority, err
}
