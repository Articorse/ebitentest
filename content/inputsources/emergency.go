package inputsources

import (
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/pathfinding"
	"ebittest/utils"
)

// What a colonist drops everything for, the more urgent the higher
type EmergencyKind uint8

const (
	Emergency_None EmergencyKind = iota
	// Out of what finishing the site on Tile would shut it in
	Emergency_GetOut
	// To Tile, the closest with air that's safe to breathe, or failing that, that can be breathed
	Emergency_GetToAir
)

type Emergency struct {
	Kind EmergencyKind
	// What it's about
	Tile utils.Vec2i
	// When it's over at the latest, 0 for an emergency that lasts until it's dealt with
	Until uint64
}

// A job waiting on an emergency. Other colonists leave its stops alone while it's claimed, and once it isn't, they can
// take them over, which it then can't carry on with.
type PausedJob struct {
	State      ColonistState
	Stops      []JobStop
	Acted      bool
	Subject    utils.Vec2i
	Score      int
	Order      uint64
	ClaimUntil uint64
	Claimed    bool
}

type emergencyLimits struct {
	ticks, claimTicks uint64
}

var emergencyLimitsOf = map[EmergencyKind]emergencyLimits{
	Emergency_GetOut:   {data.ColonistGetOutTicks, data.ColonistGetOutClaimTicks},
	Emergency_GetToAir: {0, data.ColonistGetToAirClaimTicks},
}

// Whether the entity is a colonist dealing with an emergency, which is shown over it
func InEmergency(e common.EntityId, ecsContainer *ecs.ECSContainer) bool {
	in, ok := ecsContainer.Inputs.Get(e)
	if !ok {
		return false
	}
	p, ok := in.Params().(*InputColonistParams)
	return ok && p != nil && p.Emergency.Kind != Emergency_None
}

// An emergency only gives way to a more urgent one, so that being told again doesn't put off when it ends
func (p *InputColonistParams) raise(kind EmergencyKind, tile utils.Vec2i, tick uint64) {
	if p.Emergency.Kind >= kind {
		return
	}
	limits := emergencyLimitsOf[kind]
	if p.Paused == nil && p.State != Colonist_Idle {
		p.Paused = &PausedJob{
			State: p.State, Stops: p.Stops, Acted: p.Acted, Subject: p.Subject, Score: p.Score, Order: p.Order,
			ClaimUntil: tick + limits.claimTicks, Claimed: true,
		}
	}
	p.State, p.Stops, p.Acted, p.path = Colonist_Idle, nil, false, nil
	p.Emergency = Emergency{Kind: kind, Tile: tile}
	if limits.ticks > 0 {
		p.Emergency.Until = tick + limits.ticks
	}
}

// False once the emergency's over, when it carries on with its job
func (p *InputColonistParams) handleEmergency(
	e common.EntityId,
	selfPos utils.Vec2f,
	constructor *ecs.Constructor,
	tick uint64,
	ecsContainer *ecs.ECSContainer,
) (ecs.InputState, bool) {
	if p.Paused != nil && tick >= p.Paused.ClaimUntil {
		p.Paused.Claimed = false
	}
	over := p.Emergency.Until != 0 && tick >= p.Emergency.Until
	switch p.Emergency.Kind {
	case Emergency_GetOut:
		_, onSite := ecsContainer.Terrain().Site(p.Emergency.Tile)
		over = over || !onSite
	case Emergency_GetToAir:
		air, ok := ecsContainer.AirOf(e)
		over = over || (ok && ecs.SafeAir(air)) || ecsContainer.AirTile(selfPos) == p.Emergency.Tile
	}
	if over {
		p.Emergency, p.path = Emergency{}, nil
		p.resume(e, utils.WorldPosToTilePos(selfPos), constructor, tick, ecsContainer)
		return ecs.InputState{}, false
	}
	if p.Emergency.Kind == Emergency_GetToAir {
		input, ok := p.getToAir(e, selfPos, ecsContainer)
		if !ok {
			p.Emergency, p.path = Emergency{}, nil
			p.resume(e, utils.WorldPosToTilePos(selfPos), constructor, tick, ecsContainer)
			return ecs.InputState{}, false
		}
		return input, true
	}
	return p.getOut(e, selfPos, ecsContainer), true
}

// Each time the air changes, a colonist breathing air that's bad for it is told to get to the closest air that's safe
// to breathe, or failing that, that can be breathed, if it can get to any
func (p *InputColonistParams) checkAir(e common.EntityId, selfPos utils.Vec2f, tick uint64, ecsContainer *ecs.ECSContainer) {
	if tick%data.AirStepTicks != 0 || p.Emergency.Kind >= Emergency_GetToAir || !ecsContainer.InBadAir(e) {
		return
	}
	terrain := ecsContainer.Terrain()
	self, walkSpeed := utils.WorldPosToTilePos(selfPos), ecsContainer.WalkSpeedOf(e)
	for _, good := range []func(climate.Air) bool{ecs.SafeAir, ecs.BreathableAir} {
		isGoal := func(t utils.Vec2i) bool {
			air, ok := terrain.AirAt(t)
			return ok && good(air)
		}
		path, ok := pathfinding.FindClosest(self, isGoal, data.PathMaxExpandedTiles, walkSpeed)
		if !ok {
			continue
		}
		if len(path) > 0 {
			p.raise(Emergency_GetToAir, path[len(path)-1], tick)
		}
		return
	}
}

// Along the way to the air it's been told to get to, false once there's no way there any more
func (p *InputColonistParams) getToAir(e common.EntityId, selfPos utils.Vec2f, ecsContainer *ecs.ECSContainer) (ecs.InputState, bool) {
	input := p.emptyHand(e, ecsContainer)
	walkSpeed := ecsContainer.WalkSpeedOf(e)
	if p.path == nil {
		self, air := utils.WorldPosToTilePos(selfPos), p.Emergency.Tile
		path, _, ok := pathfinding.FindTowards(self, air, 0, func(t utils.Vec2i) bool { return t == air }, data.PathMaxExpandedTiles, walkSpeed)
		if !ok {
			return input, false
		}
		p.follow(self, path)
	}
	move, _ := p.moveTo(selfPos, utils.TilePosToWorldPos(p.goal), walkSpeed)
	input.Analog1X, input.Analog1Y = move.Analog1X, move.Analog1Y
	return input, true
}

// To the closest tile round the site, off every site, that finishing it wouldn't shut it in on, where it waits
func (p *InputColonistParams) getOut(e common.EntityId, selfPos utils.Vec2f, ecsContainer *ecs.ECSContainer) ecs.InputState {
	input := p.emptyHand(e, ecsContainer)
	self, site := utils.WorldPosToTilePos(selfPos), p.Emergency.Tile
	if !onASite(e, self, ecsContainer) {
		if _, enclosed := ecsContainer.Enclosure(e, self, site, data.ColonistEnclosedTiles); !enclosed {
			p.path = nil
			return keepOffSites(e, selfPos, input, ecsContainer)
		}
	}
	walkSpeed := ecsContainer.WalkSpeedOf(e)
	if p.path == nil {
		r := 2 + bodyRadius(e, self, ecsContainer)
		round := map[utils.Vec2i]bool{}
		for y := site.Y - r; y <= site.Y+r; y++ {
			for x := site.X - r; x <= site.X+r; x++ {
				t := utils.Vec2i{X: x, Y: y}
				if !onASite(e, t, ecsContainer) && walkSpeed(t) > 0 {
					round[t] = true
				}
			}
		}
		goals := roomySpots(e, site, round, ecsContainer)
		path, _, ok := pathfinding.FindTowards(self, site, r, func(t utils.Vec2i) bool { return goals[t] }, data.PathMaxExpandedTiles, walkSpeed)
		if !ok {
			return input
		}
		p.follow(self, path)
	}
	move, _ := p.moveTo(selfPos, utils.TilePosToWorldPos(p.goal), walkSpeed)
	input.Analog1X, input.Analog1Y = move.Analog1X, move.Analog1Y
	return input
}

// Unless it's no longer worth doing, or someone else has taken it over since it stopped being claimed
func (p *InputColonistParams) resume(e common.EntityId, self utils.Vec2i, constructor *ecs.Constructor, tick uint64, ecsContainer *ecs.ECSContainer) {
	j := p.Paused
	p.Paused = nil
	p.NextPlanTick = tick
	if j == nil {
		return
	}
	p.State, p.Stops, p.Acted, p.Subject, p.Score, p.Order = j.State, j.Stops, j.Acted, j.Subject, j.Score, j.Order
	if othersClaims(e, p.Colony, ecsContainer).overlap(j, ecsContainer) {
		p.drop()
		return
	}
	p.advance(e, self, constructor, ecsContainer)
}

// Whether others have taken over any of the job
func (c claims) overlap(j *PausedJob, ecsContainer *ecs.ECSContainer) bool {
	for i, s := range j.Stops {
		if i == 0 && c.taken[s.Tile] {
			return true
		}
		switch s.Action {
		case Stop_PickUp:
			if (i > 0 || !j.Acted) && c.unclaimedStack(s.Tile, ecsContainer) <= 0 {
				return true
			}
		case Stop_Deliver:
			if site, def, ok := siteAt(s.Tile, ecsContainer); ok && c.unclaimedNeed(s.Tile, s.Kind, site, def) <= 0 {
				return true
			}
		}
	}
	return false
}
