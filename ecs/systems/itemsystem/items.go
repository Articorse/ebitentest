package itemsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"log"
)

type collector struct {
	e   common.EntityId
	pos utils.Vec2f
}

// Tossed items land, and nearby ones are pulled in, before anything picks them up. Collectors and items are in entity
// order, so that when two could pull or pick up the same item, the same one always does.
func Tick(ecsContainer *ecs.ECSContainer) {
	// Collected first, since a toss that ends takes its component away
	tossed := []common.EntityId{}
	for e := range ecs.Query1(&ecsContainer.Tosses, ecs.Without(&ecsContainer.Disabled)) {
		tossed = append(tossed, e)
	}
	for _, e := range tossed {
		ecsContainer.TickToss(e)
	}

	for _, item := range ecs.Query1(&ecsContainer.Items, ecs.Without(&ecsContainer.Disabled)) {
		item.TickPickupDelay()
	}

	collectors := []collector{}
	for e := range ecs.Query1(&ecsContainer.Inventories, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of entity %d to pick up items: %v", e, err)
			continue
		}
		collectors = append(collectors, collector{e, pos})
	}

	pullItems(collectors, ecsContainer)
	pickUpItems(collectors, ecsContainer)
}

// Towards the closest collector in reach that could pick it up, and only while one is
func pullItems(collectors []collector, ecsContainer *ecs.ECSContainer) {
	for e, item := range ecs.Query1(&ecsContainer.Items, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
		if ecsContainer.Parent(e) != -1 {
			item.StopPull()
			continue
		}

		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			log.Printf("error getting world position of item %d: %v", e, err)
			continue
		}

		closest, closestDist2 := -1, float64(data.ItemPullReachPx*data.ItemPullReachPx)
		for i, c := range collectors {
			if !ecsContainer.CanPickUp(c.e, e) {
				continue
			}
			if d := c.pos.Subtract(pos); d.X*d.X+d.Y*d.Y <= closestDist2 && (closest == -1 || d.X*d.X+d.Y*d.Y < closestDist2) {
				closest, closestDist2 = i, d.X*d.X+d.Y*d.Y
			}
		}

		if closest == -1 {
			item.StopPull()
			continue
		}
		ecsContainer.PullItem(e, collectors[closest].pos)
	}
}

func pickUpItems(collectors []collector, ecsContainer *ecs.ECSContainer) {
	type pickup struct{ collector, item common.EntityId }
	pickups := []pickup{}
	for _, c := range collectors {
		for e := range ecs.Query1(&ecsContainer.Items, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
			if !ecsContainer.CanPickUp(c.e, e) {
				continue
			}

			itemPos, err := ecsContainer.WorldPos(e)
			if err != nil {
				log.Printf("error getting world position of item %d: %v", e, err)
				continue
			}
			if d := itemPos.Subtract(c.pos); d.X*d.X+d.Y*d.Y <= data.ItemPickupReachPx*data.ItemPickupReachPx {
				pickups = append(pickups, pickup{c.e, e})
			}
		}
	}

	// Picked up after looking, since picking up disables items, which the query above goes through
	for _, p := range pickups {
		if _, err := ecsContainer.PickUpItem(p.collector, p.item); err != nil {
			log.Printf("error picking up item %d for entity %d: %v", p.item, p.collector, err)
		}
	}
}
