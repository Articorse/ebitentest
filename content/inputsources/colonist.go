package inputsources

import (
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
)

// Saved with what it's doing and where, so that a loaded colonist carries on as it was
type InputColonistParams struct {
	// Whose sites it works on, only inside its bounds
	Colony common.EntityId
	State  ColonistState
	// The site it's working on or delivering to
	Target utils.Vec2i
	// Where it's headed to reach the target from
	Stand utils.Vec2i
	// When it next looks for work, while it has none, or for building, while it's delivering
	NextPlanTick uint64

	walk
	// What it last found no way to build or deliver to, for each, which isn't looked for again until something changes,
	// since that searches all the colony it can walk to. Not saved, so a loaded colonist looks once.
	fruitless [2]*planKey
}

// Where it is isn't part of it, since walking can't take it anywhere it couldn't reach before
type planKey struct {
	sightVersion       uint64
	construct, deliver []utils.Vec2i
}

func (k *planKey) equal(o planKey) bool {
	return k != nil && k.sightVersion == o.sightVersion &&
		slices.Equal(k.construct, o.construct) && slices.Equal(k.deliver, o.deliver)
}

func (InputColonistParams) IsInputParams() {}

func (p *InputColonistParams) Copy() ecs.InputParams {
	if p == nil {
		return p
	}
	c := *p
	c.path = slices.Clone(p.path)
	c.fruitless = [2]*planKey{}
	return &c
}

// Builds the closest site on the colony's home that has everything delivered, and otherwise delivers to the closest
// that it has all that's still needed for, going by walking distance rather than as the crow flies, so that a site it can't get
// to doesn't hold it up. It keeps to a site until it's done or no longer needs it, looking for building now and then
// while delivering. It acts from where it's in reach and sees the site, but never from on it, since a tile can't be
// finished with someone standing in it. What it uses goes on the first hotbar slot as it uses it, and comes off it to
// build with an empty hand.
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
	if _, inReach := ecsContainer.AimedTileInReach(selfPos, target, constructor.RangeTiles()); inReach && self != p.Target {
		p.path = nil
		return p.use(entityId, target, ecsContainer), nil
	}

	input, ok := p.moveTo(selfPos, utils.TilePosToWorldPos(p.Stand), ecsContainer.Terrain())
	if !ok {
		p.State, p.NextPlanTick = Colonist_Idle, tick
	}
	return input, nil
}

func (p *InputColonistParams) stillWorthIt(e common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	site, def, ok := siteAt(p.Target, ecsContainer)
	switch {
	case !ok || !ecsContainer.Terrain().IsHome(p.Target):
		return false
	case p.State == Colonist_Construct:
		return site.Supplied(def)
	default:
		return !site.Supplied(def) && canSupply(e, site, def, ecsContainer)
	}
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

// Building comes first, and delivering only while there's none, each to the site closest to walk to
func (p *InputColonistParams) plan(e common.EntityId, self utils.Vec2i, rangeTiles int, ecsContainer *ecs.ECSContainer) {
	first, last, err := ecsContainer.ColonyBounds(p.Colony)
	if err != nil {
		p.State = Colonist_Idle
		return
	}

	construct, deliver := []utils.Vec2i{}, []utils.Vec2i{}
	for _, t := range ecsContainer.Terrain().SitesIn(first, last) {
		site, def, ok := siteAt(t, ecsContainer)
		switch {
		case !ok || !ecsContainer.Terrain().IsHome(t):
		case site.Supplied(def):
			construct = append(construct, t)
		case canSupply(e, site, def, ecsContainer):
			deliver = append(deliver, t)
		}
	}

	sightTerrain, versioned := ecsContainer.Terrain().(ecs.SightTerrain)
	key := planKey{construct: construct, deliver: deliver}
	if versioned {
		key.sightVersion = sightTerrain.SightVersion()
	}

	for i, job := range []struct {
		state ColonistState
		sites []utils.Vec2i
	}{{Colonist_Construct, construct}, {Colonist_Deliver, deliver}} {
		if p.State == Colonist_Deliver && job.state == Colonist_Deliver {
			return
		}
		if versioned && p.fruitless[i].equal(key) {
			continue
		}
		if target, stand, path, ok := nearestSite(self, job.sites, rangeTiles, first, last, ecsContainer); ok {
			p.State, p.Target, p.Stand = job.state, target, stand
			p.fruitless = [2]*planKey{}
			p.follow(self, path)
			return
		}
		p.fruitless[i] = &key
	}
	p.State = Colonist_Idle
}

// The tiles around each site that it can be worked on from, and the first of them reached walking out from self
func nearestSite(
	self utils.Vec2i,
	sites []utils.Vec2i,
	rangeTiles int,
	first, last utils.Vec2i,
	ecsContainer *ecs.ECSContainer,
) (site, stand utils.Vec2i, path []utils.Vec2i, ok bool) {
	if len(sites) == 0 {
		return utils.Vec2i{}, utils.Vec2i{}, nil, false
	}
	terrain := ecsContainer.Terrain()
	inColony := func(t utils.Vec2i) bool {
		return t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y
	}
	standFor := map[utils.Vec2i]utils.Vec2i{}
	for _, s := range sites {
		for dy := -rangeTiles; dy <= rangeTiles; dy++ {
			for dx := -rangeTiles; dx <= rangeTiles; dx++ {
				t := utils.Vec2i{X: s.X + dx, Y: s.Y + dy}
				if _, taken := standFor[t]; taken || t == s || !inColony(t) || terrain.WalkSpeedPercent(t) <= 0 {
					continue
				}
				if _, sees := ecsContainer.AimedTileInReach(utils.TilePosToWorldPos(t), utils.TilePosToWorldPos(s), rangeTiles); sees {
					standFor[t] = s
				}
			}
		}
	}

	isGoal := func(t utils.Vec2i) bool {
		_, ok := standFor[t]
		return ok
	}
	path, ok = pathfinding.FindNearest(self, first, last, isGoal, data.ColonistSearchMaxTiles, terrain.WalkSpeedPercent)
	if !ok {
		return utils.Vec2i{}, utils.Vec2i{}, nil, false
	}
	stand = self
	if len(path) > 0 {
		stand = path[len(path)-1]
	}
	return standFor[stand], stand, path, true
}

// What's needed goes on the first hotbar slot and is delivered all at once; to build, the slot is emptied
func (p *InputColonistParams) use(e common.EntityId, target utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
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
