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
	"math"
	"slices"
)

// The kind of job a colonist has
type ColonistState uint8

const (
	// Stands in place, with nothing to do
	Colonist_Idle ColonistState = iota
	// Works on a site that has everything delivered
	Colonist_Construct
	// Takes what a site still needs to it, having all of it
	Colonist_Deliver
	// Fells a tree on the harvest area with its axe
	Colonist_Chop
	// Picks up what lies loose on the harvest area
	Colonist_Gather
	// Puts what it carries down on storage, apart from what it works with and what sites need
	Colonist_Store
	// Takes what a site still needs off storage, and then to the site
	Colonist_Fetch
	// Takes what lies on a site off it onto storage, or nearby without room there, since the site can't be finished
	// under it
	Colonist_Haul
)

type StopAction uint8

const (
	Stop_Build StopAction = iota
	Stop_Chop
	Stop_PickUp
	Stop_Deliver
	// Onto storage
	Stop_PutDown
	// What lies on a site, onto the ground nearby
	Stop_MoveOff
)

// One step of a job, on one tile
type JobStop struct {
	Action StopAction
	Tile   utils.Vec2i
	Kind   data.ItemEnum
	// How many it picks up, delivers or puts down, which other colonists leave to it
	Count int
}

// Saved with its job, so that a loaded colonist carries on as it was
type InputColonistParams struct {
	// Whose sites it works on, only inside its bounds
	Colony common.EntityId
	State  ColonistState
	// What's left of its job, the first being what it's doing now
	Stops []JobStop
	// Whether it's done what's on the ground for the current stop, which happens in the tick it's asked for
	Acted bool
	// Where it's headed to reach the current stop from
	Stand utils.Vec2i
	// When it next looks for work, while it has none
	NextPlanTick uint64
	// The tree, site, stack or storage its job is about
	Subject utils.Vec2i
	// What its last job was and what it was about, which what it does next follows on from
	Last        ColonistState
	LastSubject utils.Vec2i
	// What its job won out with, to show how jobs are weighed up
	Score int

	walk
	// What it last found no work it could get to among, which isn't looked through again until something changes,
	// since a path that's never found is searched for until the search gives up. Not saved, so a loaded colonist looks
	// once.
	fruitless *planKey
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
	c.Stops = slices.Clone(p.Stops)
	c.path = slices.Clone(p.path)
	c.fruitless = nil
	return &c
}

// With no job, it clears what lies on the closest site on the colony's home first, then builds the closest site that
// has everything delivered, otherwise delivers to the closest that it has all that's still needed for, otherwise
// fetches what it lacks for one from storage and delivers it, otherwise picks up what lies closest on the harvest area,
// and otherwise fells the closest tree there, going by walking distance among the few closest as the crow flies, so
// that one it can't get to doesn't hold it up. With none of that to do, it puts what it carries that no site needs down
// on the closest storage. It sees to its job until it's done or can't be done any more, and only then looks for
// another. What other colonists' jobs are going to pick up, deliver or put down, and the trees and sites they're
// felling, building or clearing, are left to them. It acts from where it's in reach and sees the stop's tile, but never
// from on it, since a tile can't be finished with someone standing in it. What it uses goes on the first hotbar slot
// as it uses it, and comes off it to build with an empty hand.
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

	if p.advance(entityId, self, constructor, ecsContainer) {
		p.NextPlanTick = tick
	}
	if p.State == Colonist_Idle && tick >= p.NextPlanTick {
		p.NextPlanTick = tick + data.ColonistPlanEveryTicks
		p.plan(entityId, self, constructor, ecsContainer)
	}

	if p.State == Colonist_Idle {
		return p.stepOffSite(entityId, selfPos, ecsContainer), nil
	}

	stop := p.Stops[0]
	target := utils.TilePosToWorldPos(stop.Tile)
	if _, inReach := ecsContainer.AimedTileInReach(selfPos, target, stopRange(stop, constructor)); inReach && self != stop.Tile {
		p.path = nil
		return p.use(entityId, self, target, ecsContainer), nil
	}
	// What's built after it chose where to stand can hide the stop from there
	if self == p.Stand && !p.headFor(entityId, self, constructor, ecsContainer) {
		p.drop()
		p.NextPlanTick = tick
		return p.emptyHand(entityId, ecsContainer), nil
	}

	input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.Stand), ecsContainer.Terrain())
	if !ok {
		p.drop()
		p.NextPlanTick = tick
	}
	return input, nil
}

// Standing still, unless it's on a site, which can't be finished with it there, so it goes to the closest tile that isn't
// one
func (p *InputColonistParams) stepOffSite(e common.EntityId, selfPos utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
	terrain := ecsContainer.Terrain()
	self := utils.WorldPosToTilePos(selfPos)
	if _, onSite := terrain.Site(self); !onSite {
		p.path = nil
		return input
	}
	best, found := utils.Vec2i{}, false
	r := data.ColonistStepOffRangeTiles
	for y := self.Y - r; y <= self.Y+r; y++ {
		for x := self.X - r; x <= self.X+r; x++ {
			t := utils.Vec2i{X: x, Y: y}
			if _, onSite := terrain.Site(t); onSite || terrain.WalkSpeedPercent(t) <= 0 {
				continue
			}
			if !found || utils.Dist2(t, self) < utils.Dist2(best, self) {
				best, found = t, true
			}
		}
	}
	if !found {
		return input
	}
	move, _ := p.moveTo(selfPos, utils.TilePosToWorldPos(best), terrain)
	input.Analog1X, input.Analog1Y = move.Analog1X, move.Analog1Y
	return input
}

func stopRange(stop JobStop, constructor *ecs.Constructor) int {
	switch stop.Action {
	case Stop_Chop:
		return data.ColonistChopRangeTiles
	case Stop_PickUp, Stop_PutDown, Stop_MoveOff:
		return data.ItemReachTiles
	}
	return constructor.RangeTiles()
}

func (p *InputColonistParams) drop() {
	if p.State != Colonist_Idle {
		p.Last, p.LastSubject = p.State, p.Subject
	}
	p.State, p.Stops, p.Acted, p.path = Colonist_Idle, nil, false, nil
}

// Moves on past the stops that are done, and drops the job once what's left of it can't be done. Whether the job ended.
func (p *InputColonistParams) advance(e common.EntityId, self utils.Vec2i, constructor *ecs.Constructor, ecsContainer *ecs.ECSContainer) bool {
	for p.State != Colonist_Idle {
		if len(p.Stops) == 0 || !p.valid(e, ecsContainer) {
			p.drop()
			return true
		}
		stop := p.Stops[0]
		if !p.done(e, stop, ecsContainer) {
			return false
		}
		p.Stops, p.Acted = p.Stops[1:], false
		if stop.Action == Stop_PickUp {
			p.settle(e, stop.Kind, ecsContainer)
		}
		if len(p.Stops) == 0 || !p.headFor(e, self, constructor, ecsContainer) {
			p.drop()
			return true
		}
	}
	return false
}

func (p *InputColonistParams) done(e common.EntityId, stop JobStop, ecsContainer *ecs.ECSContainer) bool {
	switch stop.Action {
	case Stop_PickUp, Stop_PutDown, Stop_MoveOff:
		return p.Acted
	case Stop_Deliver:
		site, def, _ := siteAt(stop.Tile, ecsContainer)
		return site.Needs(stop.Kind, def) <= 0
	}
	return false
}

// Once it's picked up all it's going to of a kind, what it delivers or puts down of it is what it got, which is less
// than it set out for when someone else took some first
func (p *InputColonistParams) settle(e common.EntityId, kind data.ItemEnum, ecsContainer *ecs.ECSContainer) {
	if slices.ContainsFunc(p.Stops, func(s JobStop) bool { return s.Action == Stop_PickUp && s.Kind == kind }) {
		return
	}
	got := carried(e, kind, ecsContainer)
	for i, s := range p.Stops {
		if s.Kind == kind && (s.Action == Stop_Deliver || s.Action == Stop_PutDown) {
			p.Stops[i].Count = min(s.Count, got)
		}
	}
}

// Where it can reach the current stop from, and the path there
func (p *InputColonistParams) headFor(e common.EntityId, self utils.Vec2i, constructor *ecs.Constructor, ecsContainer *ecs.ECSContainer) bool {
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		return false
	}
	stop := p.Stops[0]
	avoid := othersClaims(e, p.Colony, ecsContainer).stands
	stand, path, _, ok := walkTo(self, stop.Tile, stopRange(stop, constructor), avoid, first, last, ecsContainer)
	if ok {
		p.Stand = stand
		p.follow(self, path)
	}
	return ok
}

// The current stop can still be done, and every site it's yet to deliver to still wants it, so that nothing's fetched
// for a site that no longer needs it
func (p *InputColonistParams) valid(e common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	terrain := ecsContainer.Terrain()
	for _, s := range p.Stops {
		if site, def, ok := siteAt(s.Tile, ecsContainer); s.Action == Stop_Deliver && (!ok || !terrain.InArea(data.Area_Home, s.Tile) || site.Needs(s.Kind, def) <= 0) {
			return false
		}
	}

	stop := p.Stops[0]
	_, onSite := terrain.Site(stop.Tile)
	switch stop.Action {
	case Stop_Build:
		site, def, ok := siteAt(stop.Tile, ecsContainer)
		return ok && terrain.InArea(data.Area_Home, stop.Tile) && site.Supplied(def)
	case Stop_Chop:
		_, hasAxe := axeSlot(e, ecsContainer)
		return hasAxe && isTree(stop.Tile, ecsContainer) && terrain.InArea(data.Area_Harvest, stop.Tile)
	case Stop_PickUp:
		if p.Acted {
			return true
		}
		stack, ok := terrain.GroundStack(stop.Tile)
		if !ok || stack.Unique != nil || stack.Kind != stop.Kind || ecsContainer.RoomFor(e, stop.Kind) <= 0 {
			return false
		}
		switch p.State {
		case Colonist_Gather:
			return gatherable(e, stop.Tile, ecsContainer)
		case Colonist_Fetch:
			return terrain.InArea(data.Area_Storage, stop.Tile)
		default:
			return onSite && terrain.InArea(data.Area_Home, stop.Tile)
		}
	case Stop_Deliver:
		site, def, _ := siteAt(stop.Tile, ecsContainer)
		return carried(e, stop.Kind, ecsContainer) >= min(stop.Count, site.Needs(stop.Kind, def))
	case Stop_PutDown:
		_, count := p.putDownFrom(e, stop, ecsContainer)
		return p.Acted || (count > 0 && terrain.InArea(data.Area_Storage, stop.Tile) && ecsContainer.GroundRoomFor(stop.Tile, stop.Kind) > 0)
	case Stop_MoveOff:
		_, onGround := terrain.GroundStack(stop.Tile)
		return p.Acted || (onSite && onGround && terrain.InArea(data.Area_Home, stop.Tile))
	}
	return false
}

// The slot it puts down from and how many, which when storing is only what no site on home needs
func (p *InputColonistParams) putDownFrom(e common.EntityId, stop JobStop, ecsContainer *ecs.ECSContainer) (slot, count int) {
	slot, ok := slotOf(e, stop.Kind, ecsContainer)
	if !ok {
		return -1, 0
	}
	inv, _ := ecsContainer.Inventories.Get(e)
	item, _ := inv.Slot(slot)
	comp, _ := ecsContainer.Items.Get(item)
	count = min(stop.Count, comp.Count())
	if p.State == Colonist_Store {
		count = min(count, carried(e, stop.Kind, ecsContainer)-p.stillNeeded(ecsContainer)[stop.Kind])
	}
	return slot, count
}

// The first slot holding a generic item of the kind
func slotOf(e common.EntityId, kind data.ItemEnum, ecsContainer *ecs.ECSContainer) (int, bool) {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return -1, false
	}
	for i := range inv.Capacity() {
		if item, ok := inv.Slot(i); ok {
			if comp, ok := ecsContainer.Items.Get(item); ok && comp.Kind() == kind && kind != data.Item_None {
				return i, true
			}
		}
	}
	return -1, false
}

// What other colonists of the colony are going to do, as their jobs' stops have it
type claims struct {
	picking    map[utils.Vec2i]int
	delivering map[siteNeed]int
	// What's going onto each tile, of one kind, as with what lies there
	storing map[utils.Vec2i]JobStop
	// The trees being felled, and sites being built or cleared, each one colonist's
	taken map[utils.Vec2i]bool
	// Where they're standing to work, so that two don't push each other off the same spot
	stands map[utils.Vec2i]bool
}

type siteNeed struct {
	tile utils.Vec2i
	kind data.ItemEnum
}

func othersClaims(e, colony common.EntityId, ecsContainer *ecs.ECSContainer) claims {
	c := claims{picking: map[utils.Vec2i]int{}, delivering: map[siteNeed]int{}, storing: map[utils.Vec2i]JobStop{}, taken: map[utils.Vec2i]bool{}, stands: map[utils.Vec2i]bool{}}
	for other, in := range ecs.Query1(&ecsContainer.Inputs) {
		q, ok := in.Params().(*InputColonistParams)
		if other == e || !ok || q == nil || q.Colony != colony {
			continue
		}
		if q.State != Colonist_Idle {
			c.stands[q.Stand] = true
		}
		for i, s := range q.Stops {
			switch s.Action {
			case Stop_PickUp:
				c.picking[s.Tile] += s.Count
			case Stop_Deliver:
				c.delivering[siteNeed{s.Tile, s.Kind}] += s.Count
			case Stop_PutDown:
				c.storing[s.Tile] = JobStop{Kind: s.Kind, Count: c.storing[s.Tile].Count + s.Count}
			}
			if i == 0 && (s.Action == Stop_Build || s.Action == Stop_Chop || s.Action == Stop_MoveOff || q.State == Colonist_Haul) {
				c.taken[s.Tile] = true
			}
		}
	}
	return c
}

func (c claims) unclaimedStack(t utils.Vec2i, ecsContainer *ecs.ECSContainer) int {
	stack, ok := ecsContainer.Terrain().GroundStack(t)
	if !ok || stack.Unique != nil {
		return 0
	}
	return stack.Count - c.picking[t]
}

func (c claims) unclaimedNeed(t utils.Vec2i, kind data.ItemEnum, site ecs.ConstructionSite, def ecs.BlueprintDef) int {
	return site.Needs(kind, def) - c.delivering[siteNeed{t, kind}]
}

func (c claims) unclaimedRoom(t utils.Vec2i, kind data.ItemEnum, ecsContainer *ecs.ECSContainer) int {
	going, ok := c.storing[t]
	if ok && going.Kind != kind {
		return 0
	}
	return ecsContainer.GroundRoomFor(t, kind) - going.Count
}

// It has all of what's still needed that no one else is bringing, not just some
func (c claims) canSupply(e common.EntityId, t utils.Vec2i, site ecs.ConstructionSite, def ecs.BlueprintDef, ecsContainer *ecs.ECSContainer) bool {
	any := false
	for _, cost := range def.Costs() {
		need := c.unclaimedNeed(t, cost.Kind, site, def)
		if need > carried(e, cost.Kind, ecsContainer) {
			return false
		}
		any = any || need > 0
	}
	return any
}

func (c claims) deliveries(t utils.Vec2i, site ecs.ConstructionSite, def ecs.BlueprintDef) []JobStop {
	stops := []JobStop{}
	for _, cost := range def.Costs() {
		if need := c.unclaimedNeed(t, cost.Kind, site, def); need > 0 {
			stops = append(stops, JobStop{Action: Stop_Deliver, Tile: t, Kind: cost.Kind, Count: need})
		}
	}
	return stops
}

// The first slot holding something of a kind, which is anything it doesn't work with, of which it has more than the
// sites on home still need between them, and how many more, which is what it puts away, keeping the rest so that what
// it fetched for a site it can't get to isn't put back straight away
func (p *InputColonistParams) storable(e common.EntityId, ecsContainer *ecs.ECSContainer) (kind data.ItemEnum, count int, ok bool) {
	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return data.Item_None, 0, false
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
			return comp.Kind(), spare, true
		}
	}
	return data.Item_None, 0, false
}

// Where on storage it can be put that no one else is filling up: the tiles with a stack of it that has room, and only
// while there are none, the empty ones, so that what's stored is kept together
func (c claims) storeSpots(kind data.ItemEnum, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) []utils.Vec2i {
	onStacks, empty := []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range storage {
		if c.unclaimedRoom(t, kind, ecsContainer) <= 0 {
			continue
		}
		_, going := c.storing[t]
		if _, taken := ecsContainer.Terrain().GroundStack(t); taken || going {
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

// Hauling comes first, then building, delivering, fetching, gathering, felling and storing, each only while there's
// none of what comes before it, and each to what's closest to walk to, leaving what other colonists are seeing to alone
func (p *InputColonistParams) plan(e common.EntityId, self utils.Vec2i, constructor *ecs.Constructor, ecsContainer *ecs.ECSContainer) {
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		p.drop()
		return
	}
	terrain := ecsContainer.Terrain()
	c := othersClaims(e, p.Colony, ecsContainer)

	haul, construct, deliver, home := []utils.Vec2i{}, []utils.Vec2i{}, []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range terrain.SitesIn(first, last) {
		if !terrain.InArea(data.Area_Home, t) {
			continue
		}
		home = append(home, t)
		if _, onGround := terrain.GroundStack(t); onGround && !c.taken[t] {
			haul = append(haul, t)
		}
		site, def, ok := siteAt(t, ecsContainer)
		switch {
		case !ok:
		case site.Supplied(def):
			if !c.taken[t] {
				construct = append(construct, t)
			}
		case c.canSupply(e, t, site, def, ecsContainer):
			deliver = append(deliver, t)
		}
	}

	chop := []utils.Vec2i{}
	if _, hasAxe := axeSlot(e, ecsContainer); hasAxe {
		for _, t := range terrain.FeaturesIn(first, last, data.Feature_Tree) {
			if terrain.InArea(data.Area_Harvest, t) && !c.taken[t] {
				chop = append(chop, t)
			}
		}
	}

	gather := slices.DeleteFunc(gatherables(e, first, last, ecsContainer), func(t utils.Vec2i) bool {
		return c.unclaimedStack(t, ecsContainer) <= 0
	})
	storage := terrain.AreaTiles(data.Area_Storage, first, last)
	fetch := []utils.Vec2i{}
	fetchFor, fetching := c.fetchSite(e, self, home, storage, ecsContainer)
	if fetching {
		lacking := c.lacking(e, fetchFor, ecsContainer)
		for _, t := range storage {
			if stack, ok := terrain.GroundStack(t); ok && lacking[stack.Kind] > 0 && c.unclaimedStack(t, ecsContainer) > 0 {
				fetch = append(fetch, t)
			}
		}
	}
	store := []utils.Vec2i{}
	if kind, _, ok := p.storable(e, ecsContainer); ok {
		store = c.storeSpots(kind, storage, ecsContainer)
	}

	jobs := []struct {
		state      ColonistState
		targets    []utils.Vec2i
		rangeTiles int
		weight     int
	}{
		{Colonist_Haul, haul, data.ItemReachTiles, data.ColonistHaulWeight},
		{Colonist_Construct, construct, constructor.RangeTiles(), data.ColonistBuildWeight},
		{Colonist_Deliver, deliver, constructor.RangeTiles(), data.ColonistDeliverWeight},
		{Colonist_Fetch, fetch, data.ItemReachTiles, data.ColonistFetchWeight},
		{Colonist_Gather, gather, data.ItemReachTiles, data.ColonistGatherWeight},
		{Colonist_Chop, chop, data.ColonistChopRangeTiles, data.ColonistChopWeight},
		{Colonist_Store, store, data.ItemReachTiles, data.ColonistStoreWeight},
	}
	sightTerrain, versioned := terrain.(ecs.SightTerrain)
	key := planKey{}
	for i, job := range jobs {
		key.targets[i] = job.targets
	}
	if versioned {
		key.sightVersion = sightTerrain.SightVersion()
		if p.fruitless.equal(key) {
			p.drop()
			return
		}
	}

	// Each as good as it could be, walking to it in a straight line, so that once the best found is better than the
	// next of these, none of the rest can beat it
	type candidate struct {
		job, best int
		target    utils.Vec2i
	}
	candidates := []candidate{}
	for i, job := range jobs {
		for _, t := range job.targets {
			best := job.weight + p.bonus(job.state, t) - walkCost(leastWalk(self, t, job.rangeTiles))
			candidates = append(candidates, candidate{job: i, best: best, target: t})
		}
	}
	slices.SortStableFunc(candidates, func(a, b candidate) int { return cmp.Compare(b.best, a.best) })

	chosen, found := struct {
		state   ColonistState
		stops   []JobStop
		subject utils.Vec2i
		stand   utils.Vec2i
		path    []utils.Vec2i
		score   int
	}{}, false
	searched := 0
	for _, cand := range candidates {
		if (found && cand.best <= chosen.score) || searched == data.ColonistPathSearches {
			break
		}
		job := jobs[cand.job]
		stops := p.stopsFor(e, job.state, cand.target, fetchFor, storage, first, last, c, ecsContainer)
		if len(stops) == 0 {
			continue
		}
		searched++
		stand, path, cost, ok := walkTo(self, cand.target, job.rangeTiles, c.stands, first, last, ecsContainer)
		score := job.weight + p.bonus(job.state, cand.target) - walkCost(cost)
		if !ok || (found && score <= chosen.score) {
			continue
		}
		subject := cand.target
		if job.state == Colonist_Fetch {
			subject = fetchFor
		}
		chosen.state, chosen.stops, chosen.subject, chosen.stand, chosen.path, chosen.score = job.state, stops, subject, stand, path, score
		found = true
	}
	if !found {
		p.fruitless = &key
		p.drop()
		return
	}
	p.State, p.Stops, p.Acted, p.Stand = chosen.state, chosen.stops, false, chosen.stand
	p.Subject, p.Score = chosen.subject, chosen.score
	p.fruitless = nil
	p.follow(self, chosen.path)
}

// For carrying on with the same work, and more for what naturally follows the last job: gathering what a tree felled
// nearby dropped, gathering more nearby and then putting it away, and building a site it just supplied or cleared
func (p *InputColonistParams) bonus(state ColonistState, target utils.Vec2i) int {
	bonus := 0
	if state == p.Last {
		bonus += data.ColonistSameWorkBonus
	}
	r := data.ColonistFollowUpRangeTiles
	near := utils.Dist2(target, p.LastSubject) <= r*r
	switch {
	case state == Colonist_Gather && (p.Last == Colonist_Chop || p.Last == Colonist_Gather) && near,
		state == Colonist_Store && p.Last == Colonist_Gather,
		state == Colonist_Construct && target == p.LastSubject && (p.Last == Colonist_Deliver || p.Last == Colonist_Fetch || p.Last == Colonist_Haul):
		bonus += data.ColonistFollowUpBonus
	}
	return bonus
}

// What walking a path costing that much takes off a job's score
func walkCost(pathCost int) int {
	return pathCost * data.ColonistWalkCostPercent / 100
}

// What a path to somewhere in reach of the target costs at least, walking a straight line over ground walked at full
// speed
func leastWalk(self, target utils.Vec2i, rangeTiles int) int {
	d := math.Sqrt(float64(utils.Dist2(self, target))) - float64(rangeTiles)*math.Sqrt2
	return int(max(d, 0) * pathfinding.StraightStepCost)
}

// What the job on the target takes, and how much of what it picks up, delivers or puts down
func (p *InputColonistParams) stopsFor(
	e common.EntityId,
	state ColonistState,
	target, fetchFor utils.Vec2i,
	storage []utils.Vec2i,
	first, last utils.Vec2i,
	c claims,
	ecsContainer *ecs.ECSContainer,
) []JobStop {
	switch state {
	case Colonist_Haul:
		return c.haulStops(e, target, storage, ecsContainer)
	case Colonist_Construct:
		return []JobStop{{Action: Stop_Build, Tile: target}}
	case Colonist_Deliver:
		site, def, _ := siteAt(target, ecsContainer)
		return c.deliveries(target, site, def)
	case Colonist_Fetch:
		return c.fetchStops(e, fetchFor, target, storage, ecsContainer)
	case Colonist_Gather:
		return c.gatherStops(e, target, first, last, ecsContainer)
	case Colonist_Chop:
		return []JobStop{{Action: Stop_Chop, Tile: target}}
	case Colonist_Store:
		kind, spare, _ := p.storable(e, ecsContainer)
		n := min(spare, c.unclaimedRoom(target, kind, ecsContainer))
		return []JobStop{{Action: Stop_PutDown, Tile: target, Kind: kind, Count: n}}
	}
	return nil
}

// Carried off onto the storage closest to the site while there's room there, as much as it can carry and there's
// room for, and otherwise moved off onto the ground nearby, which is also where what's never put away, a unique item,
// goes
func (c claims) haulStops(e common.EntityId, site utils.Vec2i, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) []JobStop {
	stack, _ := ecsContainer.Terrain().GroundStack(site)
	spots := c.storeSpots(stack.Kind, storage, ecsContainer)
	if n := min(stack.Count, ecsContainer.RoomFor(e, stack.Kind)); stack.Unique == nil && n > 0 && len(spots) > 0 {
		to := slices.MinFunc(spots, func(a, b utils.Vec2i) int { return cmp.Compare(utils.Dist2(a, site), utils.Dist2(b, site)) })
		n = min(n, c.unclaimedRoom(to, stack.Kind, ecsContainer))
		return []JobStop{
			{Action: Stop_PickUp, Tile: site, Kind: stack.Kind, Count: n},
			{Action: Stop_PutDown, Tile: to, Kind: stack.Kind, Count: n},
		}
	}
	return []JobStop{{Action: Stop_MoveOff, Tile: site}}
}

// The stack on the target and then the others of its kind closest to it, as many as it can carry
func (c claims) gatherStops(e common.EntityId, target, first, last utils.Vec2i, ecsContainer *ecs.ECSContainer) []JobStop {
	stack, _ := ecsContainer.Terrain().GroundStack(target)
	r := data.ColonistGatherRangeTiles
	from := utils.Vec2i{X: max(target.X-r, first.X), Y: max(target.Y-r, first.Y)}
	to := utils.Vec2i{X: min(target.X+r, last.X), Y: min(target.Y+r, last.Y)}
	tiles := slices.DeleteFunc(gatherables(e, from, to, ecsContainer), func(t utils.Vec2i) bool {
		on, _ := ecsContainer.Terrain().GroundStack(t)
		return on.Kind != stack.Kind || utils.Dist2(t, target) > r*r
	})
	slices.SortStableFunc(tiles, func(a, b utils.Vec2i) int { return cmp.Compare(utils.Dist2(a, target), utils.Dist2(b, target)) })

	stops := []JobStop{}
	room := ecsContainer.RoomFor(e, stack.Kind)
	for _, t := range tiles {
		if n := min(c.unclaimedStack(t, ecsContainer), room); n > 0 {
			stops = append(stops, JobStop{Action: Stop_PickUp, Tile: t, Kind: stack.Kind, Count: n})
			room -= n
		}
	}
	return stops
}

// How many of each kind it lacks for what's still needed at the site that no one else is bringing
func (c claims) lacking(e common.EntityId, t utils.Vec2i, ecsContainer *ecs.ECSContainer) map[data.ItemEnum]int {
	lacking := map[data.ItemEnum]int{}
	site, def, ok := siteAt(t, ecsContainer)
	if !ok {
		return lacking
	}
	for _, cost := range def.Costs() {
		if n := c.unclaimedNeed(t, cost.Kind, site, def) - carried(e, cost.Kind, ecsContainer); n > 0 {
			lacking[cost.Kind] = n
		}
	}
	return lacking
}

// The site on home closest as the crow flies that it lacks something for, which there's enough of on storage that no
// one else is taking to make up for, and that it can carry
func (c claims) fetchSite(e common.EntityId, self utils.Vec2i, sites, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) (utils.Vec2i, bool) {
	stored := map[data.ItemEnum]int{}
	for _, t := range storage {
		if stack, ok := ecsContainer.Terrain().GroundStack(t); ok && stack.Unique == nil {
			stored[stack.Kind] += c.unclaimedStack(t, ecsContainer)
		}
	}
	best, found := utils.Vec2i{}, false
	for _, t := range sites {
		lacking := c.lacking(e, t, ecsContainer)
		enough := len(lacking) > 0
		for kind, n := range lacking {
			enough = enough && n <= stored[kind] && n <= ecsContainer.RoomFor(e, kind)
		}
		if enough && (!found || utils.Dist2(t, self) < utils.Dist2(best, self)) {
			best, found = t, true
		}
	}
	return best, found
}

// Picking up what it lacks from the storage it's headed for first, then from what's closest to that, and delivering
// all that the site still needs that no one else is bringing
func (c claims) fetchStops(e common.EntityId, site, from utils.Vec2i, storage []utils.Vec2i, ecsContainer *ecs.ECSContainer) []JobStop {
	lacking := c.lacking(e, site, ecsContainer)
	tiles := slices.Clone(storage)
	slices.SortStableFunc(tiles, func(a, b utils.Vec2i) int { return cmp.Compare(utils.Dist2(a, from), utils.Dist2(b, from)) })
	stops := []JobStop{}
	for _, t := range tiles {
		stack, ok := ecsContainer.Terrain().GroundStack(t)
		if !ok || stack.Unique != nil {
			continue
		}
		if n := min(lacking[stack.Kind], c.unclaimedStack(t, ecsContainer)); n > 0 {
			stops = append(stops, JobStop{Action: Stop_PickUp, Tile: t, Kind: stack.Kind, Count: n})
			lacking[stack.Kind] -= n
		}
	}
	s, def, _ := siteAt(site, ecsContainer)
	return append(stops, c.deliveries(site, s, def)...)
}

// What lies on the ground is handled without a hand. What's delivered, or the axe, goes on the first hotbar slot, and
// what's needed is delivered all at once; to build, the slot is emptied.
func (p *InputColonistParams) use(e common.EntityId, self utils.Vec2i, target utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
	stop := p.Stops[0]
	switch stop.Action {
	case Stop_PickUp:
		input.PickUp = ecs.ItemPickUp{PickUp: true, Tile: stop.Tile, Count: stop.Count}
		p.Acted = true
		return input
	case Stop_PutDown:
		slot, count := p.putDownFrom(e, stop, ecsContainer)
		input.PutDown = ecs.ItemPutDown{PutDown: true, From: slot, Count: count, Tile: stop.Tile}
		p.Acted = true
		return input
	case Stop_MoveOff:
		stack, _ := ecsContainer.Terrain().GroundStack(stop.Tile)
		if to, ok := haulTo(self, stop.Tile, stack, ecsContainer); ok {
			input.MoveOnGround = ecs.GroundMove{Move: true, From: stop.Tile, To: to}
			p.Acted = true
		}
		return input
	case Stop_Chop:
		if slot, ok := axeSlot(e, ecsContainer); ok {
			input.Inventory = ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: slot, To: 0}
		}
	case Stop_Deliver:
		input.Inventory = ecs.InventoryAction{}
		if slot, ok := slotOf(e, stop.Kind, ecsContainer); ok {
			input.Inventory = ecs.InventoryAction{Kind: ecs.InventoryAction_Bind, From: slot, To: 0}
		}
		input.All = true
	}
	input.AimPos, input.HasAim = target, true
	input.MainHandEqAbility1 = 1
	return input
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

// Somewhere the target can be worked on from, the path there and what it costs. A path is given up on after
// data.PathMaxExpandedTiles, so a target far enough round a maze can be missed.
func walkTo(
	self, target utils.Vec2i,
	rangeTiles int,
	avoid map[utils.Vec2i]bool,
	first, last utils.Vec2i,
	ecsContainer *ecs.ECSContainer,
) (stand utils.Vec2i, path []utils.Vec2i, cost int, ok bool) {
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
	spots := standSpots(target, rangeTiles, inColony, avoid, ecsContainer)
	path, cost, ok = pathfinding.FindTowards(self, target, rangeTiles, func(t utils.Vec2i) bool { return spots[t] }, data.PathMaxExpandedTiles, speed)
	if !ok {
		return utils.Vec2i{}, nil, 0, false
	}
	stand = self
	if len(path) > 0 {
		stand = path[len(path)-1]
	}
	return stand, path, cost, true
}

// The walkable tiles within reach that see the site, other than the site itself. Other sites, where standing would
// hold up their building, and the spots to avoid are only stood on when there's nowhere else.
func standSpots(site utils.Vec2i, rangeTiles int, inColony func(utils.Vec2i) bool, avoid map[utils.Vec2i]bool, ecsContainer *ecs.ECSContainer) map[utils.Vec2i]bool {
	spots, better := map[utils.Vec2i]bool{}, map[utils.Vec2i]bool{}
	for dy := -rangeTiles; dy <= rangeTiles; dy++ {
		for dx := -rangeTiles; dx <= rangeTiles; dx++ {
			t := utils.Vec2i{X: site.X + dx, Y: site.Y + dy}
			if t == site || !inColony(t) || ecsContainer.Terrain().WalkSpeedPercent(t) <= 0 {
				continue
			}
			if _, sees := ecsContainer.AimedTileInReach(utils.TilePosToWorldPos(t), utils.TilePosToWorldPos(site), rangeTiles); sees {
				spots[t] = true
				if _, onSite := ecsContainer.Terrain().Site(t); !onSite && !avoid[t] {
					better[t] = true
				}
			}
		}
	}
	if len(better) > 0 {
		return better
	}
	return spots
}

// What's needed, or the axe, goes on the first hotbar slot, and what's needed is delivered all at once; to build, the
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
