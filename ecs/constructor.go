package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/fov"
	"ebittest/utils"
	"fmt"
	"math"
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

// The tile under the aim, and whether it's within rangeTiles of a tile the entity's body takes up and in sight from
// there, so that a body bigger than a tile reaches as far past its edge as one of a tile does
func (x *ECSContainer) AimedTileInReach(e common.EntityId, aim utils.Vec2f, rangeTiles int) (utils.Vec2i, bool) {
	aimed := utils.WorldPosToTilePos(aim)
	pos, err := x.WorldPos(e)
	if err != nil {
		return aimed, false
	}
	return aimed, x.TileInReachFrom(x.BodyTiles(e, utils.WorldPosToTilePos(pos)), aimed, rangeTiles)
}

// Whether the tile is within rangeTiles of one of the tiles a body takes up, and in sight from it
func (x *ECSContainer) TileInReachFrom(body []utils.Vec2i, t utils.Vec2i, rangeTiles int) bool {
	for _, from := range body {
		if utils.Dist2(from, t) > rangeTiles*rangeTiles {
			continue
		}
		inSight := false
		fov.Compute(from, rangeTiles, x.terrain.IsOpaqueTile, func(seen utils.Vec2i) {
			inSight = inSight || seen == t
		})
		if inSight {
			return true
		}
	}
	return false
}

// The tiles the entity's body takes up standing in the middle of the tile, as far as its physics collider reaches past
// it, which is the tile alone for a body that fits in one
func (x *ECSContainer) BodyTiles(e common.EntityId, at utils.Vec2i) []utils.Vec2i {
	lo, hi := x.bodyReach(e)
	tiles := make([]utils.Vec2i, 0, (hi.X-lo.X+1)*(hi.Y-lo.Y+1))
	for dy := lo.Y; dy <= hi.Y; dy++ {
		for dx := lo.X; dx <= hi.X; dx++ {
			tiles = append(tiles, utils.Vec2i{X: at.X + dx, Y: at.Y + dy})
		}
	}
	return tiles
}

// How far the tiles the entity's body takes up reach past the tile it's in the middle of in the direction, so that what
// it does at a reach from itself starts from where its reach is measured from. 0 for a body within its tile.
func (x *ECSContainer) BodyOverhang(e common.EntityId, dir utils.Vec2f) float64 {
	lo, hi := x.bodyReach(e)
	dir = dir.Normalized()
	exit := func(lo, hi utils.Vec2i) float64 {
		dist := math.Inf(1)
		axes := []struct{ d, lo, hi float64 }{{dir.X, float64(lo.X), float64(hi.X)}, {dir.Y, float64(lo.Y), float64(hi.Y)}}
		for _, axis := range axes {
			switch {
			case axis.d > 0:
				dist = math.Min(dist, (axis.hi*data.TileSize+data.TileSize/2)/axis.d)
			case axis.d < 0:
				dist = math.Min(dist, (-axis.lo*data.TileSize+data.TileSize/2)/-axis.d)
			}
		}
		return dist
	}
	return exit(lo, hi) - exit(utils.Vec2i{}, utils.Vec2i{})
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
	tile, ok := x.AimedTileInReach(e, aim, c.rangeTiles)
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

// Works on the roof job over the tile once the constructor's ready, if the tile's within its reach
func (x *ECSContainer) WorkOnRoof(e common.EntityId, t utils.Vec2i) {
	c, ok := x.Constructors.Get(e)
	if !ok || x.TickIdx < c.readyTick {
		return
	}
	pos, err := x.WorldPos(e)
	if err != nil || !x.TileInReachFrom(x.BodyTiles(e, utils.WorldPosToTilePos(pos)), t, c.rangeTiles) {
		return
	}
	if worked, _ := x.terrain.WorkOnRoof(t, c.work); worked {
		c.readyTick = x.TickIdx + uint64(c.cooldownMs/data.TickMs)
	}
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
