package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
	"fmt"
)

// Lets an entity deliver materials to construction sites and build them, with what's in hand and an empty hand
// respectively
type Constructor struct {
	rangeTiles int
	// Added to a site with each strike
	work       int
	cooldownMs int
	// The first tick it can act again, rather than a countdown, so that nothing has to tick it
	readyTick uint64
}

func NewConstructorComponent(rangeTiles, work, cooldownMs int) *Constructor {
	return &Constructor{rangeTiles: rangeTiles, work: work, cooldownMs: cooldownMs}
}

func (x *Constructor) RangeTiles() int {
	return x.rangeTiles
}

// The tile under the aim, if it's within the range of the tile at from and can be seen from there
func (x *ECSContainer) AimedTileInReach(from, aim utils.Vec2f, rangeTiles int) (utils.Vec2i, bool) {
	origin := utils.WorldPosToTilePos(from)
	aimed := utils.WorldPosToTilePos(aim)
	if utils.Dist2(origin, aimed) > rangeTiles*rangeTiles {
		return utils.Vec2i{}, false
	}

	inSight := false
	fov.Compute(origin, rangeTiles, x.terrain.IsOpaqueTile, func(t utils.Vec2i) {
		if t == aimed {
			inSight = true
		}
	})
	return aimed, inSight
}

// Delivers what's in the constructor's hand to the construction site under its aim, if the site needs it, one at a time
// or all that's needed at once; with an empty hand, it works on the site once everything is delivered, though not the
// last strike while something is in the way of the tile. Handled is whether the aim is on a site it can do either for,
// cooldown or not, so that what's in hand isn't also used on it.
func (x *ECSContainer) Construct(e common.EntityId, aim utils.Vec2f, all bool) (handled bool, err error) {
	c, ok := x.Constructors.Get(e)
	if !ok {
		return false, nil
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return false, fmt.Errorf("error getting the world position of entity %d to construct: %w", e, err)
	}
	tile, ok := x.AimedTileInReach(pos, aim, c.rangeTiles)
	if !ok {
		return false, nil
	}
	site, ok := x.terrain.Site(tile)
	if !ok {
		return false, nil
	}
	def, ok := x.blueprintAtlas.Def(site.Blueprint)
	if !ok {
		return false, nil
	}

	held, holding := x.heldBy(e)
	if holding {
		item, ok := x.Items.Get(held)
		needed := 0
		if ok {
			needed = site.Needs(item.kind, def)
		}
		if needed <= 0 {
			return false, nil
		}
		if x.TickIdx < c.readyTick {
			return true, nil
		}

		count := 1
		if all {
			count = item.count
		}
		taken := x.terrain.DeliverToSite(tile, item.kind, count)
		for range taken {
			if err := x.ConsumeItem(held); err != nil {
				return true, fmt.Errorf("error using up item %d delivered to the site at %v: %w", held, tile, err)
			}
		}
		c.readyTick = x.TickIdx + uint64(c.cooldownMs/data.TickMs)
		return true, nil
	}

	if !site.Supplied(def) {
		return false, nil
	}
	if x.TickIdx < c.readyTick {
		return true, nil
	}
	home := x.terrain.InArea(data.Area_Home, tile)
	worked, built := x.terrain.BuildSite(tile, c.work, !x.IsTileOccupied(tile))
	if worked {
		c.readyTick = x.TickIdx + uint64(c.cooldownMs/data.TickMs)
	}
	if built && home {
		x.spreadHome(tile)
	}
	return true, nil
}

// Whether the entity's next go at building the site on the tile would finish it
func (x *ECSContainer) FinishesSite(e common.EntityId, t utils.Vec2i) bool {
	c, ok := x.Constructors.Get(e)
	if !ok {
		return false
	}
	site, ok := x.terrain.Site(t)
	if !ok {
		return false
	}
	def, ok := x.blueprintAtlas.Def(site.Blueprint)
	return ok && site.Supplied(def) && site.Health+SiteWorkStep(def, site.MaxHealth, c.work) >= site.MaxHealth
}

// The item on the selected hotbar slot, whether or not it fits in a hand
func (x *ECSContainer) heldBy(e common.EntityId) (common.EntityId, bool) {
	inv, ok := x.Inventories.Get(e)
	if !ok {
		return -1, false
	}
	return inv.HotbarItem(inv.Selected())
}

func (x Constructor) Copy() Constructor {
	return x
}

func (x Constructor) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Constructors.addComponent(e, x.Copy())
}

type constructorDto struct {
	RangeTiles int
	Work       int
	CooldownMs int
	ReadyTick  uint64
}

func (x Constructor) toDto() (ComponentDto, error) {
	return &constructorDto{RangeTiles: x.rangeTiles, Work: x.work, CooldownMs: x.cooldownMs, ReadyTick: x.readyTick}, nil
}

func (x constructorDto) toComponent() (Component, error) {
	return &Constructor{rangeTiles: x.RangeTiles, work: x.Work, cooldownMs: x.CooldownMs, readyTick: x.ReadyTick}, nil
}
