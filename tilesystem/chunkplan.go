package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"errors"
	"fmt"
	"slices"
)

type chunkPlan struct {
	// Chunks within a loader's radius, which must be active
	activate []utils.Vec2i
	// Active chunks outside this set are deactivated, which leaves a margin against flip-flopping on borders
	keepActive map[utils.Vec2i]struct{}
	// Chunks loaded ahead of time so that activation rarely has to wait, in loading order
	prefetch []utils.Vec2i
	resident map[utils.Vec2i]struct{}
}

func buildChunkPlan(ecsCont *ecs.ECSContainer) (plan chunkPlan, err error) {
	clm := ecsCont.ChunkLoaderManager

	activate := make(map[utils.Vec2i]int)
	plan.keepActive = make(map[utils.Vec2i]struct{})
	prefetch := make(map[utils.Vec2i]int)

	for _, eId := range ecsCont.ChunkLoaders.GetEntities() {
		worldPos, ierr := ecsCont.WorldPos(eId)
		if ierr != nil {
			err = errors.Join(err, fmt.Errorf("error getting world position of entity %d: %v", eId, ierr))
			continue
		}

		radius, ierr := clm.GetRadius(eId, ecsCont)
		if ierr != nil {
			err = errors.Join(err, fmt.Errorf("error getting radius of entity %d: %v", eId, ierr))
			continue
		}

		center := utils.WorldPosToChunkGridPos(worldPos)
		outer := radius + data.ChunkPrefetchMargin

		for dy := -outer; dy <= outer; dy++ {
			for dx := -outer; dx <= outer; dx++ {
				pos := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
				ring := max(utils.AbsInt(dx), utils.AbsInt(dy))
				d := dx*dx + dy*dy

				if ring <= radius {
					addWithPriority(activate, pos, d)
				}
				if ring <= radius+data.ChunkDeactivateMargin {
					plan.keepActive[pos] = struct{}{}
				}
				addWithPriority(prefetch, pos, d)
			}
		}
	}

	plan.activate = sortedByPriority(activate)
	plan.prefetch = sortedByPriority(prefetch)
	plan.resident = make(map[utils.Vec2i]struct{}, len(prefetch))
	for pos := range prefetch {
		plan.resident[pos] = struct{}{}
	}

	return plan, err
}

func addWithPriority(priorities map[utils.Vec2i]int, pos utils.Vec2i, priority int) {
	if old, ok := priorities[pos]; !ok || priority < old {
		priorities[pos] = priority
	}
}

func sortedByPriority(priorities map[utils.Vec2i]int) []utils.Vec2i {
	positions := make([]utils.Vec2i, 0, len(priorities))
	for pos := range priorities {
		positions = append(positions, pos)
	}

	slices.SortFunc(positions, func(a, b utils.Vec2i) int {
		if priorities[a] != priorities[b] {
			return priorities[a] - priorities[b]
		}
		if a.X != b.X {
			return a.X - b.X
		}
		return a.Y - b.Y
	})

	return positions
}
