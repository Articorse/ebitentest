package facingsystem

import (
	"ebittest/ecs"
	"log"
	"math"
)

func Tick(ecsContainer *ecs.ECSContainer) {
	for e, row := range ecs.Query2(&ecsContainer.Facings, &ecsContainer.Transforms, ecs.Without(&ecsContainer.Disabled)) {
		aim, ok := ecsContainer.AimOf(e)
		if !ok {
			continue
		}

		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d to face its aim: %v", e, err)
			continue
		}
		if aim == pos {
			continue
		}

		parentRot := 0.0
		if p := ecsContainer.Parent(e); p != -1 {
			if parentRot, err = ecsContainer.WorldRotation(p); err != nil {
				log.Printf("error getting world rotation of entity %d's parent to face its aim: %v", e, err)
				continue
			}
		}

		target := math.Atan2(aim.Y-pos.Y, aim.X-pos.X) - parentRot
		row.B.SetLocalRotation(row.A.Turn(row.B.LocalRotation(), target))
	}

	// A swing keeps its side until it's over, since flipping mid-swing would snap the item to the other side while
	// it's still coming back
	for e, sw := range ecs.Query1(&ecsContainer.Swings, ecs.Without(&ecsContainer.Disabled)) {
		aim, ok := ecsContainer.AimOf(e)
		if !ok || ecsContainer.IsSwinging(e) {
			continue
		}

		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d to pick its swing's side: %v", e, err)
			continue
		}
		sw.SetMirrored(aim.X < pos.X)
	}
}
