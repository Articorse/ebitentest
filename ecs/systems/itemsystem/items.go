package itemsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"log"
)

// Tossed items land before anything picks them up. Collectors and items are in entity order, so that when two could
// pick up the same item, the same one always does.
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

	type pickup struct{ collector, item common.EntityId }
	pickups := []pickup{}
	for collector := range ecs.Query1(&ecsContainer.Inventories, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
		collectorPos, err := ecsContainer.WorldPos(collector)
		if err != nil {
			log.Printf("error getting world position of entity %d to pick up items: %v", collector, err)
			continue
		}

		for e, item := range ecs.Query1(&ecsContainer.Items, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
			if ecsContainer.Parent(e) != -1 || !item.CanBePickedUp() {
				continue
			}

			itemPos, err := ecsContainer.WorldPos(e)
			if err != nil {
				log.Printf("error getting world position of item %d: %v", e, err)
				continue
			}
			if d := itemPos.Subtract(collectorPos); d.X*d.X+d.Y*d.Y <= data.ItemPickupReachPx*data.ItemPickupReachPx {
				pickups = append(pickups, pickup{collector, e})
			}
		}
	}

	// Picked up after looking, since picking up disables items, which the queries above go through
	for _, p := range pickups {
		if _, err := ecsContainer.PickUpItem(p.collector, p.item); err != nil {
			log.Printf("error picking up item %d for entity %d: %v", p.item, p.collector, err)
		}
	}
}
