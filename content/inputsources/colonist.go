package inputsources

import (
	"cmp"
	"ebittest/content/abilities"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/pathfinding"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"
)

type ColonistState uint8

const (
	// Stands in place, with nothing to do
	Colonist_Idle ColonistState = iota
	// Works on a site that has everything delivered
	Colonist_Construct
	// Takes what a site still needs to it
	Colonist_Deliver
	// Fells a tree on the harvest area with its axe
	Colonist_Chop
	// Picks up what lies loose on the harvest area
	Colonist_Gather
	// Puts what it carries in a storage area, apart from what it works with and what sites need
	Colonist_Store
	// Takes what a site still needs out of a storage area, to deliver it
	Colonist_Fetch
)

// Saved with what it's doing and where, so that a loaded colonist carries on as it was
type InputColonistParams struct {
	// Whose sites it works on, only inside its bounds
	Colony common.EntityId
	State  ColonistState
	// The site it's working on or delivering to, the tree it's felling, or where what it's gathering was
	Target utils.Vec2i
	// What it's gathering
	Item common.EntityId
	// Where it's putting what it carries, or taking what a site needs from
	Storage common.EntityId
	// The site it's fetching for
	Site utils.Vec2i
	// Where it's headed to reach the target from
	Stand utils.Vec2i
	// When it next looks for work, while it has none, or for building, while it's delivering
	NextPlanTick uint64

	walk
	// What it last found no way to build, deliver to, fetch from, gather, fell or store in, for each, which isn't looked for again until something
	// changes, since a path that's never found is searched for until the search gives up. Not saved, so a loaded
	// colonist looks once.
	fruitless [6]*planKey
}

// Where it is isn't part of it, since walking can't take it anywhere it couldn't reach before
type planKey struct {
	sightVersion                                   uint64
	construct, deliver, fetch, gather, chop, store []utils.Vec2i
}

func (k *planKey) equal(o planKey) bool {
	return k != nil && k.sightVersion == o.sightVersion && slices.Equal(k.construct, o.construct) &&
		slices.Equal(k.deliver, o.deliver) && slices.Equal(k.fetch, o.fetch) && slices.Equal(k.gather, o.gather) &&
		slices.Equal(k.chop, o.chop) && slices.Equal(k.store, o.store)
}

func (InputColonistParams) IsInputParams() {}

func (p *InputColonistParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.path = slices.Clone(p.path)
	c.fruitless = [6]*planKey{}
	return &c
}

// Builds the closest site on the colony's home that has everything delivered, otherwise delivers to the closest that
// it has all that's still needed for, otherwise fetches what it lacks for one from storage, otherwise picks up what
// lies closest on the harvest area, and otherwise fells the closest tree there, going by walking distance among the few
// closest as the crow flies, so that one it can't get to doesn't hold it up. With none of that to do, it puts away
// what it carries that no site needs in the closest storage area with room. It keeps to its
// target until it's done or no longer worth it, looking for something more important now and then. It acts from where
// it's in reach and sees the target, but never from on a site, since a tile can't be finished with someone standing in
// it. What it uses goes on the first hotbar slot as it uses it, and comes off it to build with an empty hand.
func ColonistInputSource(
	entityId common.EntityId,
	tick uint64,
	params ecs.InputParams,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, error) {
	p, ok := params.(*InputColonistParams)
	if !ok || p == nil {
		return ecs.InputState{}, fmt.Errorf("colonist input source called with invalid params: %T", params)
	}
	constructor, ok := ecsContainer.Constructors.Get(entityId)
	if !ok {
		return ecs.InputState{}, fmt.Errorf("colonist %d can't build", entityId)
	}
	selfPos, err := ecsContainer.WorldPos(entityId)
	if err != nil {
		return ecs.InputState{}, fmt.Errorf("error getting world position of colonist %d: %w", entityId, err)
	}
	self := utils.WorldPosToTilePos(selfPos)

	if p.State != Colonist_Idle && !p.stillWorthIt(entityId, ecsContainer) {
		p.State = Colonist_Idle
		p.NextPlanTick = tick
	}
	if p.State != Colonist_Construct && tick >= p.NextPlanTick {
		p.NextPlanTick = tick + data.ColonistPlanEveryTicks
		p.plan(entityId, self, constructor.RangeTiles(), ecsContainer)
	}

	if p.State == Colonist_Idle {
		p.path = nil
		return p.emptyHand(entityId, ecsContainer), nil
	}

	if p.State == Colonist_Gather {
		return p.gather(entityId, selfPos, tick, ecsContainer), nil
	}
	if p.State == Colonist_Store || p.State == Colonist_Fetch {
		return p.useStorage(entityId, selfPos, tick, ecsContainer), nil
	}
	target := utils.TilePosToWorldPos(p.Target)
	if _, inReach := ecsContainer.AimedTileInReach(selfPos, target, p.rangeTiles(constructor)); inReach && self != p.Target {
		p.path = nil
		return p.use(entityId, target, ecsContainer), nil
	}

	input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.Stand), ecsContainer.Terrain())
	if !ok {
		p.State, p.NextPlanTick = Colonist_Idle, tick
	}
	return input, nil
}

// Up to the item and picking it up, after which it looks for more straight away, so that it gathers what's lying
// around before moving on
func (p *InputColonistParams) gather(e common.EntityId, selfPos utils.Vec2f, tick uint64, ecsContainer *ecs.ECSContainer) ecs.InputState {
	itemPos, err := ecsContainer.WorldPos(p.Item)
	if err != nil {
		p.State, p.NextPlanTick = Colonist_Idle, tick
		return p.emptyHand(e, ecsContainer)
	}
	if d := itemPos.Subtract(selfPos); d.X*d.X+d.Y*d.Y <= data.ItemPickUpOnPurposeReachPx*data.ItemPickUpOnPurposeReachPx {
		p.path = nil
		input := p.emptyHand(e, ecsContainer)
		input.PickUp = ecs.ItemPickUp{PickUp: true, Item: p.Item}
		return input
	}
	input, ok := p.moveTo(selfPos, itemPos, ecsContainer.Terrain())
	if !ok {
		p.State, p.NextPlanTick = Colonist_Idle, tick
	}
	return input
}

// From within reach of the storage area, standing anywhere, even in it, it puts away a slot at a time, or takes out
// what the site lacks a kind at a time
func (p *InputColonistParams) useStorage(e common.EntityId, selfPos utils.Vec2f, tick uint64, ecsContainer *ecs.ECSContainer) ecs.InputState {
	target := utils.TilePosToWorldPos(p.Target)
	if _, inReach := ecsContainer.AimedTileInReach(selfPos, target, data.StoreRangeTiles); inReach {
		p.path = nil
		input := p.emptyHand(e, ecsContainer)
		if p.State == Colonist_Fetch {
			if kind, count, ok := toFetch(e, p.Site, p.Storage, ecsContainer); ok {
				input.Take = ecs.ItemTaking{Take: true, Kind: kind, Count: count, Storage: p.Storage}
			}
		} else if slot, count, ok := p.storableSlot(e, p.Storage, ecsContainer); ok {
			input.Store = ecs.ItemStoring{Store: true, From: slot, Count: count, Storage: p.Storage}
		}
		return input
	}
	input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.Stand), ecsContainer.Terrain())
	if !ok {
		p.State, p.NextPlanTick = Colonist_Idle, tick
	}
	return input
}

// The first slot holding something it doesn't work with, which is anything of a kind, that there's room for in the
// storage area, and how much of it no site on home still needs, which it keeps, so that what it fetched for a site it
// can't get to isn't put back straight away
func (p *InputColonistParams) storableSlot(e, storage common.EntityId, ecsContainer *ecs.ECSContainer) (slot, count int, ok bool) {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok || !ecsContainer.StorageAreas.HasComponent(storage) {
		return -1, 0, false
	}
	needed := p.stillNeeded(ecsContainer)
	for i := range inv.Capacity() {
		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		comp, ok := ecsContainer.Items.Get(item)
		if !ok || comp.Kind() == data.Item_None || !ecsContainer.HasRoomFor(storage, item) {
			continue
		}
		if spare := carried(e, comp.Kind(), ecsContainer) - needed[comp.Kind()]; spare > 0 {
			return i, min(spare, comp.Count()), true
		}
	}
	return -1, 0, false
}

// How many of each kind the sites on the colony's home still need between them
func (p *InputColonistParams) stillNeeded(ecsContainer *ecs.ECSContainer) map[data.ItemEnum]int {
	needed := map[data.ItemEnum]int{}
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		return needed
	}
	for _, t := range ecsContainer.Terrain().SitesIn(first, last) {
		site, def, ok := siteAt(t, ecsContainer)
		if !ok || !ecsContainer.Terrain().InArea(data.Area_Home, t) {
			continue
		}
		for _, c := range def.Costs() {
			needed[c.Kind] += site.Needs(c.Kind, def)
		}
	}
	return needed
}

// The first kind it lacks for the site that's in the storage area, and how many it lacks
func toFetch(e common.EntityId, siteTile utils.Vec2i, storage common.EntityId, ecsContainer *ecs.ECSContainer) (data.ItemEnum, int, bool) {
	site, def, ok := siteAt(siteTile, ecsContainer)
	if !ok {
		return data.Item_None, 0, false
	}
	for _, c := range def.Costs() {
		if lacking := site.Needs(c.Kind, def) - carried(e, c.Kind, ecsContainer); lacking > 0 && stored(storage, c.Kind, ecsContainer) > 0 {
			return c.Kind, lacking, true
		}
	}
	return data.Item_None, 0, false
}

func stored(storage common.EntityId, kind data.ItemEnum, ecsContainer *ecs.ECSContainer) int {
	for _, c := range ecsContainer.StorageContents(storage) {
		if c.Kind == kind {
			return c.Count
		}
	}
	return 0
}

// The closest tile of each storage area in the colony that has a use, and the area of each
func storageTargets(self, first, last utils.Vec2i, useful func(storage common.EntityId) bool, ecsContainer *ecs.ECSContainer) ([]utils.Vec2i, map[utils.Vec2i]common.EntityId) {
	tiles, storages := []utils.Vec2i{}, map[utils.Vec2i]common.EntityId{}
	for s, area := range ecs.Query1(&ecsContainer.StorageAreas) {
		t := area.ClosestTile(self)
		if t.X < first.X || t.X > last.X || t.Y < first.Y || t.Y > last.Y {
			continue
		}
		if useful(s) {
			tiles = append(tiles, t)
			storages[t] = s
		}
	}
	return tiles, storages
}

// The site on home closest as the crow flies that it lacks something for, which there's enough of in the colony's
// storage areas to make up for
func fetchSite(e common.EntityId, self utils.Vec2i, sites []utils.Vec2i, storages []common.EntityId, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	best, found := utils.Vec2i{}, false
	for _, t := range sites {
		site, def, ok := siteAt(t, ecsContainer)
		if !ok || site.Supplied(def) || canSupply(e, site, def, ecsContainer) {
			continue
		}
		enough := true
		for _, c := range def.Costs() {
			available := carried(e, c.Kind, ecsContainer)
			for _, s := range storages {
				available += stored(s, c.Kind, ecsContainer)
			}
			enough = enough && site.Needs(c.Kind, def) <= available
		}
		if enough && (!found || utils.Dist2(t, self) < utils.Dist2(best, self)) {
			best, found = t, true
		}
	}
	return best, found
}

func (p *InputColonistParams) rangeTiles(constructor *ecs.Constructor) int {
	if p.State == Colonist_Chop {
		return data.ColonistChopRangeTiles
	}
	return constructor.RangeTiles()
}

func (p *InputColonistParams) stillWorthIt(e common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	if p.State == Colonist_Gather {
		return gatherable(e, p.Item, ecsContainer)
	}
	if p.State == Colonist_Store {
		_, _, ok := p.storableSlot(e, p.Storage, ecsContainer)
		return ok
	}
	if p.State == Colonist_Fetch {
		site, def, ok := siteAt(p.Site, ecsContainer)
		if !ok || !ecsContainer.Terrain().InArea(data.Area_Home, p.Site) || site.Supplied(def) || canSupply(e, site, def, ecsContainer) {
			return false
		}
		_, _, ok = toFetch(e, p.Site, p.Storage, ecsContainer)
		return ok
	}
	if p.State == Colonist_Chop {
		_, hasAxe := axeSlot(e, ecsContainer)
		return hasAxe && isTree(p.Target, ecsContainer) && ecsContainer.Terrain().InArea(data.Area_Harvest, p.Target)
	}
	site, def, ok := siteAt(p.Target, ecsContainer)
	switch {
	case !ok || !ecsContainer.Terrain().InArea(data.Area_Home, p.Target):
		return false
	case p.State == Colonist_Construct:
		return site.Supplied(def)
	default:
		return !site.Supplied(def) && canSupply(e, site, def, ecsContainer)
	}
}

// Lying loose and still on the harvest area, with room for it
func gatherable(e, item common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	comp, ok := ecsContainer.Items.Get(item)
	if !ok || !slices.Contains(data.HarvestItems, comp.Kind()) || !ecsContainer.CanPickUp(e, item) {
		return false
	}
	pos, err := ecsContainer.WorldPos(item)
	return err == nil && ecsContainer.Terrain().InArea(data.Area_Harvest, utils.WorldPosToTilePos(pos))
}

// Each tile with something to gather on it, and on each the first of what's there, in entity order
func gatherables(e common.EntityId, first, last utils.Vec2i, ecsContainer *ecs.ECSContainer) ([]utils.Vec2i, map[utils.Vec2i]common.EntityId) {
	items := map[utils.Vec2i]common.EntityId{}
	for item := range ecs.Query1(&ecsContainer.Items, ecs.With(&ecsContainer.Transforms), ecs.Without(&ecsContainer.Disabled)) {
		if !gatherable(e, item, ecsContainer) {
			continue
		}
		pos, _ := ecsContainer.WorldPos(item)
		t := utils.WorldPosToTilePos(pos)
		if t.X < first.X || t.X > last.X || t.Y < first.Y || t.Y > last.Y {
			continue
		}
		if other, ok := items[t]; !ok || item < other {
			items[t] = item
		}
	}
	tiles := slices.Collect(maps.Keys(items))
	// Map order differs from run to run, and what colonists choose mustn't
	slices.SortFunc(tiles, func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return tiles, items
}

func isTree(t utils.Vec2i, ecsContainer *ecs.ECSContainer) bool {
	return len(ecsContainer.Terrain().FeaturesIn(t, t, data.Feature_Tree)) == 1
}

// The inventory slot of what it fells trees with
func axeSlot(e common.EntityId, ecsContainer *ecs.ECSContainer) (int, bool) {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return -1, false
	}
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if eq, ok := ecsContainer.Equipments.Get(item); ok && eq.HasAbility(abilities.Ability_Slash) {
				return i, true
			}
		}
	}
	return -1, false
}

func siteAt(t utils.Vec2i, ecsContainer *ecs.ECSContainer) (ecs.ConstructionSite, ecs.BlueprintDef, bool) {
	site, ok := ecsContainer.Terrain().Site(t)
	if !ok {
		return ecs.ConstructionSite{}, ecs.BlueprintDef{}, false
	}
	def, ok := ecsContainer.BlueprintAtlas().Def(site.Blueprint)
	return site, def, ok
}

// It has all that's still needed, not just some
func canSupply(e common.EntityId, site ecs.ConstructionSite, def ecs.BlueprintDef, ecsContainer *ecs.ECSContainer) bool {
	for _, c := range def.Costs() {
		if site.Needs(c.Kind, def) > carried(e, c.Kind, ecsContainer) {
			return false
		}
	}
	return true
}

func carried(e common.EntityId, kind data.ItemEnum, ecsContainer *ecs.ECSContainer) int {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return 0
	}
	n := 0
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if comp, ok := ecsContainer.Items.Get(item); ok && comp.Kind() == kind {
				n += comp.Count()
			}
		}
	}
	return n
}

// Building comes first, then delivering, fetching, gathering, felling and storing, each only while there's none of what
// comes before it, and each to what's closest to walk to
func (p *InputColonistParams) plan(e common.EntityId, self utils.Vec2i, rangeTiles int, ecsContainer *ecs.ECSContainer) {
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		p.State = Colonist_Idle
		return
	}

	construct, deliver, home := []utils.Vec2i{}, []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range ecsContainer.Terrain().SitesIn(first, last) {
		if ecsContainer.Terrain().InArea(data.Area_Home, t) {
			home = append(home, t)
		}
		site, def, ok := siteAt(t, ecsContainer)
		switch {
		case !ok || !ecsContainer.Terrain().InArea(data.Area_Home, t):
		case site.Supplied(def):
			construct = append(construct, t)
		case canSupply(e, site, def, ecsContainer):
			deliver = append(deliver, t)
		}
	}

	chop := []utils.Vec2i{}
	if _, hasAxe := axeSlot(e, ecsContainer); hasAxe {
		for _, t := range ecsContainer.Terrain().FeaturesIn(first, last, data.Feature_Tree) {
			if ecsContainer.Terrain().InArea(data.Area_Harvest, t) {
				chop = append(chop, t)
			}
		}
	}

	gather, items := gatherables(e, first, last, ecsContainer)
	allStorage := func(common.EntityId) bool { return true }
	_, colonyStorage := storageTargets(self, first, last, allStorage, ecsContainer)
	fetchFor, fetching := fetchSite(e, self, home, slices.Sorted(maps.Values(colonyStorage)), ecsContainer)
	fetch, fetchFrom := storageTargets(self, first, last, func(s common.EntityId) bool {
		_, _, ok := toFetch(e, fetchFor, s, ecsContainer)
		return fetching && ok
	}, ecsContainer)
	store, storeIn := storageTargets(self, first, last, func(s common.EntityId) bool {
		_, _, ok := p.storableSlot(e, s, ecsContainer)
		return ok
	}, ecsContainer)

	sightTerrain, versioned := ecsContainer.Terrain().(ecs.SightTerrain)
	key := planKey{construct: construct, deliver: deliver, fetch: fetch, gather: gather, chop: chop, store: store}
	if versioned {
		key.sightVersion = sightTerrain.SightVersion()
	}

	for i, job := range []struct {
		state      ColonistState
		targets    []utils.Vec2i
		rangeTiles int
	}{
		{Colonist_Construct, construct, rangeTiles},
		{Colonist_Deliver, deliver, rangeTiles},
		{Colonist_Fetch, fetch, data.StoreRangeTiles},
		{Colonist_Gather, gather, 0},
		{Colonist_Chop, chop, data.ColonistChopRangeTiles},
		{Colonist_Store, store, data.StoreRangeTiles},
	} {
		// It keeps to what it's doing when nothing more important turned up
		if p.State == job.state {
			return
		}
		if versioned && p.fruitless[i].equal(key) {
			continue
		}
		if target, stand, path, ok := nearestSite(self, job.targets, job.rangeTiles, first, last, ecsContainer); ok {
			p.State, p.Target, p.Stand, p.Item, p.Storage = job.state, target, stand, -1, -1
			if job.state == Colonist_Gather {
				p.Item = items[target]
			}
			if job.state == Colonist_Store {
				p.Storage = storeIn[target]
			}
			if job.state == Colonist_Fetch {
				p.Storage, p.Site = fetchFrom[target], fetchFor
			}
			p.fruitless = [6]*planKey{}
			p.follow(self, path)
			return
		}
		p.fruitless[i] = &key
	}
	p.State = Colonist_Idle
}

// Of the few sites, or trees, closest as the crow flies, the one cheapest to walk to somewhere it can be worked on
// from, which keeps the searching to a few paths heading for them, while one that's close but hard to get to still
// loses out to one a little further that isn't. A path is given up on after data.PathMaxExpandedTiles, so a site far enough
// round a maze can be missed.
func nearestSite(
	self utils.Vec2i,
	sites []utils.Vec2i,
	rangeTiles int,
	first, last utils.Vec2i,
	ecsContainer *ecs.ECSContainer,
) (site, stand utils.Vec2i, path []utils.Vec2i, ok bool) {
	terrain := ecsContainer.Terrain()
	inColony := func(t utils.Vec2i) bool {
		return t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y
	}
	speed := func(t utils.Vec2i) int {
		if !inColony(t) {
			return 0
		}
		return terrain.WalkSpeedPercent(t)
	}

	closest := slices.Clone(sites)
	slices.SortStableFunc(closest, func(a, b utils.Vec2i) int {
		return cmp.Compare(utils.Dist2(a, self), utils.Dist2(b, self))
	})
	// The next few are only tried when none of these can be got to
	bestCost := -1
	for from := 0; from < len(closest) && bestCost == -1; from += data.ColonistJobCandidates {
		for _, s := range closest[from:min(len(closest), from+data.ColonistJobCandidates)] {
			spots := standSpots(s, rangeTiles, inColony, ecsContainer)
			isSpot := func(t utils.Vec2i) bool { return spots[t] }
			p, cost, found := pathfinding.FindTowards(self, s, rangeTiles, isSpot, data.PathMaxExpandedTiles, speed)
			if found && (bestCost == -1 || cost < bestCost) {
				site, path, bestCost = s, p, cost
			}
		}
	}
	if bestCost == -1 {
		return utils.Vec2i{}, utils.Vec2i{}, nil, false
	}
	stand = self
	if len(path) > 0 {
		stand = path[len(path)-1]
	}
	return site, stand, path, true
}

// The walkable tiles within reach that see the site, other than the site itself, or with no reach, the site itself,
// where what's gathered lies
func standSpots(site utils.Vec2i, rangeTiles int, inColony func(utils.Vec2i) bool, ecsContainer *ecs.ECSContainer) map[utils.Vec2i]bool {
	spots := map[utils.Vec2i]bool{}
	if rangeTiles == 0 {
		if inColony(site) && ecsContainer.Terrain().WalkSpeedPercent(site) > 0 {
			spots[site] = true
		}
		return spots
	}
	for dy := -rangeTiles; dy <= rangeTiles; dy++ {
		for dx := -rangeTiles; dx <= rangeTiles; dx++ {
			t := utils.Vec2i{X: site.X + dx, Y: site.Y + dy}
			if t == site || !inColony(t) || ecsContainer.Terrain().WalkSpeedPercent(t) <= 0 {
				continue
			}
			if _, sees := ecsContainer.AimedTileInReach(utils.TilePosToWorldPos(t), utils.TilePosToWorldPos(site), rangeTiles); sees {
				spots[t] = true
			}
		}
	}
	return spots
}

// What's needed, or the axe, goes on the first hotbar slot, and what's needed is delivered all at once; to build, the
// slot is emptied
func (p *InputColonistParams) use(e common.EntityId, target utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
	if slot, ok := axeSlot(e, ecsContainer); ok && p.State == Colonist_Chop {
		input.Inventory = ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: slot, To: 0}
	}
	if p.State == Colonist_Deliver {
		input.Inventory = ecs.InventoryAction{}
		if slot, ok := p.neededSlot(e, ecsContainer); ok {
			input.Inventory = ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: slot, To: 0}
		}
		input.All = true
	}
	input.AimPos, input.HasAim = target, true
	input.MainHandEqAbility1 = 1
	return input
}

func (p *InputColonistParams) neededSlot(e common.EntityId, ecsContainer *ecs.ECSContainer) (int, bool) {
	site, def, ok := siteAt(p.Target, ecsContainer)
	inv, invOk := ecsContainer.Inventories.Get(e)
	if !ok || !invOk {
		return -1, false
	}
	for i := range inv.Capacity() {
		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		if comp, ok := ecsContainer.Items.Get(item); ok && site.Needs(comp.Kind(), def) > 0 {
			return i, true
		}
	}
	return -1, false
}

// The first hotbar slot selected and pointing at nothing
func (p *InputColonistParams) emptyHand(e common.EntityId, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := ecs.InputState{SelectSlot: 1}
	if inv, ok := ecsContainer.Inventories.Get(e); ok {
		if _, bound := inv.HotbarItem(0); bound {
			input.Inventory = ecs.InventoryAction{Kind: ecs.InventoryAction_Unbind, To: 0}
		}
	}
	return input
}
