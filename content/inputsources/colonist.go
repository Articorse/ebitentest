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
	// Puts what it carries down on storage, apart from what it works with and what sites need
	Colonist_Store
	// Takes what a site still needs off storage, to deliver it
	Colonist_Fetch
	// Takes what lies on a site off it, since the site can't be finished under it
	Colonist_Haul
)

// Saved with what it's doing and where, so that a loaded colonist carries on as it was
type InputColonistParams struct {
	// Whose sites it works on, only inside its bounds
	Colony common.EntityId
	State  ColonistState
	// The site it's working on, delivering to or hauling off, the tree it's felling, or where what it's gathering or
	// fetching lies, or where it's putting what it carries
	Target utils.Vec2i
	// The site it's fetching for
	Site utils.Vec2i
	// Where it's headed to reach the target from
	Stand utils.Vec2i
	// When it next looks for work, while it has none, or for building, while it's delivering
	NextPlanTick uint64

	walk
	// What it last found no way to get to for each job, which isn't looked for again until something changes, since a
	// path that's never found is searched for until the search gives up. Not saved, so a loaded colonist looks once.
	fruitless [7]*planKey
}

// Where it is isn't part of it, since walking can't take it anywhere it couldn't reach before
type planKey struct {
	sightVersion uint64
	// The targets of each job
	targets [7][]utils.Vec2i
}

func (k *planKey) equal(o planKey) bool {
	if k == nil || k.sightVersion != o.sightVersion {
		return false
	}
	for i := range k.targets {
		if !slices.Equal(k.targets[i], o.targets[i]) {
			return false
		}
	}
	return true
}

func (InputColonistParams) IsInputParams() {}

func (p *InputColonistParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.path = slices.Clone(p.path)
	c.fruitless = [7]*planKey{}
	return &c
}

// Clears what lies on the closest site on the colony's home first, then builds the closest that has everything
// delivered, otherwise delivers to the closest that it has all that's still needed for, otherwise fetches what it
// lacks for one from storage, otherwise picks up what lies closest on the harvest area, and otherwise fells the closest
// tree there, going by walking distance among the few closest as the crow flies, so that one it can't get to doesn't
// hold it up. With none of that to do, it puts what it carries that no site needs down on the closest storage. It keeps
// to its target until it's done or no longer worth it, looking for something more important now and then. It acts
// from where it's in reach and sees the target, but never from on it, since a tile can't be finished with someone
// standing in it. What it uses goes on the first hotbar slot as it uses it, and comes off it to build with an empty
// hand.
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

	target := utils.TilePosToWorldPos(p.Target)
	if _, inReach := ecsContainer.AimedTileInReach(selfPos, target, p.rangeTiles(constructor)); inReach && self != p.Target {
		p.path = nil
		return p.use(entityId, self, target, ecsContainer), nil
	}

	input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.Stand), ecsContainer.Terrain())
	if !ok {
		p.State, p.NextPlanTick = Colonist_Idle, tick
	}
	return input, nil
}

// The first slot holding something of a kind, which is anything it doesn't work with, of which it has more than the
// sites on home still need between them, and how many more, which is what it puts away, keeping the rest so that what
// it fetched for a site it can't get to isn't put back straight away
func (p *InputColonistParams) storable(e common.EntityId, ecsContainer *ecs.ECSContainer) (slot int, kind data.ItemEnum, count int, ok bool) {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return -1, data.Item_None, 0, false
	}
	needed := p.stillNeeded(ecsContainer)
	for i := range inv.Capacity() {
		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		comp, ok := ecsContainer.Items.Get(item)
		if !ok || comp.Kind() == data.Item_None {
			continue
		}
		if spare := carried(e, comp.Kind(), ecsContainer) - needed[comp.Kind()]; spare > 0 {
			return i, comp.Kind(), min(spare, comp.Count()), true
		}
	}
	return -1, data.Item_None, 0, false
}

// Where on storage it can be put: the tiles with a stack of it that has room, and only while there are none, the empty
// ones, so that what's stored is kept together
func storeSpots(kind data.ItemEnum, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) []utils.Vec2i {
	onStacks, empty := []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range storage {
		if ecsContainer.GroundRoomFor(t, kind) <= 0 {
			continue
		}
		if _, taken := ecsContainer.Terrain().GroundStack(t); taken {
			onStacks = append(onStacks, t)
		} else {
			empty = append(empty, t)
		}
	}
	if len(onStacks) > 0 {
		return onStacks
	}
	return empty
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

// The first kind it lacks for the site that lies on the tile, and how many it lacks
func toFetch(e common.EntityId, siteTile, from utils.Vec2i, ecsContainer *ecs.ECSContainer) (data.ItemEnum, int, bool) {
	site, def, ok := siteAt(siteTile, ecsContainer)
	stack, onGround := ecsContainer.Terrain().GroundStack(from)
	if !ok || !onGround || stack.Unique != nil {
		return data.Item_None, 0, false
	}
	for _, c := range def.Costs() {
		if lacking := site.Needs(c.Kind, def) - carried(e, c.Kind, ecsContainer); lacking > 0 && c.Kind == stack.Kind {
			return c.Kind, lacking, true
		}
	}
	return data.Item_None, 0, false
}

// The site on home closest as the crow flies that it lacks something for, which there's enough of in storage to make
// up for
func fetchSite(e common.EntityId, self utils.Vec2i, sites, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	stored := map[data.ItemEnum]int{}
	for _, t := range storage {
		if stack, ok := ecsContainer.Terrain().GroundStack(t); ok && stack.Unique == nil {
			stored[stack.Kind] += stack.Count
		}
	}
	best, found := utils.Vec2i{}, false
	for _, t := range sites {
		site, def, ok := siteAt(t, ecsContainer)
		if !ok || site.Supplied(def) || canSupply(e, site, def, ecsContainer) {
			continue
		}
		enough := true
		for _, c := range def.Costs() {
			enough = enough && site.Needs(c.Kind, def) <= carried(e, c.Kind, ecsContainer)+stored[c.Kind]
		}
		if enough && (!found || utils.Dist2(t, self) < utils.Dist2(best, self)) {
			best, found = t, true
		}
	}
	return best, found
}

// Somewhere off the site within reach of where it stands for what lies there: the closest tile to the site that can
// take it and isn't another site
func haulTo(self, site utils.Vec2i, stack ecs.GroundStack, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	r := data.ItemReachTiles
	best, found := utils.Vec2i{}, false
	for y := self.Y - r; y <= self.Y+r; y++ {
		for x := self.X - r; x <= self.X+r; x++ {
			t := utils.Vec2i{X: x, Y: y}
			if _, onSite := ecsContainer.Terrain().Site(t); onSite || utils.Dist2(t, self) > r*r {
				continue
			}
			on, taken := ecsContainer.Terrain().GroundStack(t)
			fits := !taken && ecsContainer.Terrain().WalkSpeedPercent(t) > 0
			if stack.Unique == nil {
				fits = ecsContainer.GroundRoomFor(t, stack.Kind) > 0 && (!taken || on.Kind == stack.Kind)
			}
			if fits && (!found || utils.Dist2(t, site) < utils.Dist2(best, site)) {
				best, found = t, true
			}
		}
	}
	return best, found
}

func (p *InputColonistParams) rangeTiles(constructor *ecs.Constructor) int {
	if p.State == Colonist_Chop {
		return data.ColonistChopRangeTiles
	}
	if p.State == Colonist_Gather || p.State == Colonist_Fetch || p.State == Colonist_Store || p.State == Colonist_Haul {
		return data.ItemReachTiles
	}
	return constructor.RangeTiles()
}

func (p *InputColonistParams) stillWorthIt(e common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	if p.State == Colonist_Gather {
		return gatherable(e, p.Target, ecsContainer)
	}
	if p.State == Colonist_Store {
		_, kind, _, ok := p.storable(e, ecsContainer)
		return ok && ecsContainer.Terrain().InArea(data.Area_Storage, p.Target) && ecsContainer.GroundRoomFor(p.Target, kind) > 0
	}
	if p.State == Colonist_Fetch {
		site, def, ok := siteAt(p.Site, ecsContainer)
		if !ok || !ecsContainer.Terrain().InArea(data.Area_Home, p.Site) || site.Supplied(def) || canSupply(e, site, def, ecsContainer) {
			return false
		}
		_, _, ok = toFetch(e, p.Site, p.Target, ecsContainer)
		return ok && ecsContainer.Terrain().InArea(data.Area_Storage, p.Target)
	}
	if p.State == Colonist_Haul {
		_, onSite := ecsContainer.Terrain().Site(p.Target)
		_, onGround := ecsContainer.Terrain().GroundStack(p.Target)
		return onSite && onGround && ecsContainer.Terrain().InArea(data.Area_Home, p.Target)
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

// Lying on the harvest area, and not already put away on storage, of a kind that's harvested, with room for some of it
func gatherable(e common.EntityId, t utils.Vec2i, ecsContainer *ecs.ECSContainer) bool {
	stack, ok := ecsContainer.Terrain().GroundStack(t)
	return ok && stack.Unique == nil && slices.Contains(data.HarvestItems, stack.Kind) &&
		ecsContainer.RoomFor(e, stack.Kind) > 0 && ecsContainer.Terrain().InArea(data.Area_Harvest, t) &&
		!ecsContainer.Terrain().InArea(data.Area_Storage, t)
}

func gatherables(e common.EntityId, first, last utils.Vec2i, ecsContainer *ecs.ECSContainer) []utils.Vec2i {
	return slices.DeleteFunc(ecsContainer.Terrain().GroundStacksIn(first, last), func(t utils.Vec2i) bool {
		return !gatherable(e, t, ecsContainer)
	})
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

// Hauling comes first, then building, delivering, fetching, gathering, felling and storing, each only while there's
// none of what comes before it, and each to what's closest to walk to
func (p *InputColonistParams) plan(e common.EntityId, self utils.Vec2i, rangeTiles int, ecsContainer *ecs.ECSContainer) {
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		p.State = Colonist_Idle
		return
	}

	haul, construct, deliver, home := []utils.Vec2i{}, []utils.Vec2i{}, []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range ecsContainer.Terrain().SitesIn(first, last) {
		if ecsContainer.Terrain().InArea(data.Area_Home, t) {
			home = append(home, t)
			if _, onGround := ecsContainer.Terrain().GroundStack(t); onGround {
				haul = append(haul, t)
			}
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

	gather := gatherables(e, first, last, ecsContainer)
	storage := ecsContainer.Terrain().AreaTiles(data.Area_Storage, first, last)
	fetch := []utils.Vec2i{}
	fetchFor, fetching := fetchSite(e, self, home, storage, ecsContainer)
	for _, t := range storage {
		if _, _, ok := toFetch(e, fetchFor, t, ecsContainer); fetching && ok {
			fetch = append(fetch, t)
		}
	}
	store := []utils.Vec2i{}
	if _, kind, _, ok := p.storable(e, ecsContainer); ok {
		store = storeSpots(kind, storage, ecsContainer)
	}

	jobs := []struct {
		state      ColonistState
		targets    []utils.Vec2i
		rangeTiles int
	}{
		{Colonist_Haul, haul, data.ItemReachTiles},
		{Colonist_Construct, construct, rangeTiles},
		{Colonist_Deliver, deliver, rangeTiles},
		{Colonist_Fetch, fetch, data.ItemReachTiles},
		{Colonist_Gather, gather, data.ItemReachTiles},
		{Colonist_Chop, chop, data.ColonistChopRangeTiles},
		{Colonist_Store, store, data.ItemReachTiles},
	}
	sightTerrain, versioned := ecsContainer.Terrain().(ecs.SightTerrain)
	key := planKey{}
	for i, job := range jobs {
		key.targets[i] = job.targets
	}
	if versioned {
		key.sightVersion = sightTerrain.SightVersion()
	}

	for i, job := range jobs {
		// It keeps to what it's doing when nothing more important turned up
		if p.State == job.state {
			return
		}
		if versioned && p.fruitless[i].equal(key) {
			continue
		}
		if target, stand, path, ok := nearestSite(self, job.targets, job.rangeTiles, first, last, ecsContainer); ok {
			p.State, p.Target, p.Stand = job.state, target, stand
			if job.state == Colonist_Fetch {
				p.Site = fetchFor
			}
			p.fruitless = [7]*planKey{}
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

// The walkable tiles within reach that see the site, other than the site itself
func standSpots(site utils.Vec2i, rangeTiles int, inColony func(utils.Vec2i) bool, ecsContainer *ecs.ECSContainer) map[utils.Vec2i]bool {
	spots := map[utils.Vec2i]bool{}
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
func (p *InputColonistParams) use(e common.EntityId, self utils.Vec2i, target utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
	// Handling what lies on the ground takes no hand
	switch p.State {
	case Colonist_Gather:
		input.PickUp = ecs.ItemPickUp{PickUp: true, Tile: p.Target}
		return input
	case Colonist_Fetch:
		if _, lacking, ok := toFetch(e, p.Site, p.Target, ecsContainer); ok {
			input.PickUp = ecs.ItemPickUp{PickUp: true, Tile: p.Target, Count: lacking}
		}
		return input
	case Colonist_Store:
		if slot, _, count, ok := p.storable(e, ecsContainer); ok {
			input.PutDown = ecs.ItemPutDown{PutDown: true, From: slot, Count: count, Tile: p.Target}
		}
		return input
	case Colonist_Haul:
		input.MoveOnGround, input.PickUp = p.haul(e, self, ecsContainer)
		return input
	}
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

// Carried off to be put away while there's storage with room for it, and otherwise moved off onto the ground nearby,
// which is also where what's never put away, a unique item, goes
func (p *InputColonistParams) haul(e common.EntityId, self utils.Vec2i, ecsContainer *ecs.ECSContainer) (ecs.GroundMove, ecs.ItemPickUp) {
	stack, ok := ecsContainer.Terrain().GroundStack(p.Target)
	if !ok {
		return ecs.GroundMove{}, ecs.ItemPickUp{}
	}
	if first, last, err := ecsContainer.ColonyBounds(p.Colony); err == nil && stack.Unique == nil && ecsContainer.RoomFor(e, stack.Kind) > 0 {
		if len(storeSpots(stack.Kind, ecsContainer.Terrain().AreaTiles(data.Area_Storage, first, last), ecsContainer)) > 0 {
			return ecs.GroundMove{}, ecs.ItemPickUp{PickUp: true, Tile: p.Target}
		}
	}
	if to, ok := haulTo(self, p.Target, stack, ecsContainer); ok {
		return ecs.GroundMove{Move: true, From: p.Target, To: to}, ecs.ItemPickUp{}
	}
	return ecs.GroundMove{}, ecs.ItemPickUp{}
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
