package tilesystem

import (
	"cmp"
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"maps"
	"slices"
)

// The temperature of a wall or door
type heat struct {
	temperature climate.Centicelsius
	// Heat taken in or given off short of a whole hundredth of a degree, in mJ, so that slow flows still add up
	remainder int64
}

func (x *heat) take(mJ, capacity int64) {
	if capacity <= 0 {
		return
	}
	total := x.remainder + mJ
	x.temperature += climate.Centicelsius(total / capacity)
	x.remainder = total % capacity
}

type solidHeatDto struct {
	X, Y          int
	Centicelsius  int32
	HeatRemainder int64
}

// In tile order, so that saving the same chunk twice writes the same file
func solidHeatDtos(c *chunk) []solidHeatDto {
	dtos := make([]solidHeatDto, 0, len(c.solidHeat))
	for t, h := range c.solidHeat {
		dtos = append(dtos, solidHeatDto{X: t.X, Y: t.Y, Centicelsius: int32(h.temperature), HeatRemainder: h.remainder})
	}
	slices.SortFunc(dtos, func(a, b solidHeatDto) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })
	return dtos
}

func solidHeatFromDtos(dtos []solidHeatDto) map[utils.Vec2i]*heat {
	solidHeat := make(map[utils.Vec2i]*heat, len(dtos))
	for _, d := range dtos {
		solidHeat[utils.Vec2i{X: d.X, Y: d.Y}] = &heat{temperature: climate.Centicelsius(d.Centicelsius), remainder: d.HeatRemainder}
	}
	return solidHeat
}

// The wall or door on the tile, if there's one that keeps heat
func (cc *ChunkContainer) solidThermal(t utils.Vec2i) (*thermalDef, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return nil, false
	}
	if def := cc.atlas[c.tiles[data.TileLayer_Wall][c.tileIndex(t)]]; def.thermal != nil {
		return def.thermal, true
	}
	if f, ok := c.features[t]; ok && cc.featureAtlas[f].thermal != nil {
		return cc.featureAtlas[f].thermal, true
	}
	return nil, false
}

// What it takes to warm the air by a hundredth of a degree, in mJ
func (cc *ChunkContainer) airCapacity(air climate.Air) int64 {
	capacity := int64(0)
	for g, mg := range air {
		capacity += mg * cc.gases[data.GasEnum(g)].SpecificHeat / (data.MilligramsPerGram * 100)
	}
	return capacity
}

// How well the air on a tile passes heat on, in mW per degree, in proportion to how much of each gas there is
func (cc *ChunkContainer) airConductance(perTile climate.Air) int64 {
	conductance := int64(0)
	for g, mg := range perTile {
		conductance += mg * cc.gases[data.GasEnum(g)].Conductance / data.MilligramsPerKg
	}
	return conductance
}

// Across an edge between two things, which passes heat no better than the worse of them, and none if either passes none
func edgeConductance(a, b int64) int64 {
	if a <= 0 || b <= 0 {
		return 0
	}
	return 2 * a * b / (a + b)
}

// The heat in mJ that crosses from what's at from to what's at to in a step, no more than takes them halfway to the
// same temperature, so that a step never overshoots. A capacity of 0 is one that never changes, as the outdoors'.
func heatFlow(conductance int64, from, to climate.Centicelsius, fromCapacity, toCapacity int64) int64 {
	diff := int64(from - to)
	mJ := conductance * diff * data.AirStepTicks * data.HeatTimeScale / (data.TPS * 100)
	least := fromCapacity
	if least == 0 || (toCapacity != 0 && toCapacity < least) {
		least = toCapacity
	}
	if least > 0 {
		limit := abs64(diff) * least / 2
		mJ = max(-limit, min(limit, mJ))
	}
	return mJ
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// What a room's air or a wall or door is, as heat passes between them
type heatNode struct {
	room uint64
	// The wall or door's, for one that's not a room
	solid utils.Vec2i
}

// Heat passes across every edge between a room's air and the walls and doors around it, and between those and the
// walls and doors behind them, and on to the outdoors' air, whose temperature nothing changes. Those further from a room
// than that keep none and pass none on. How well each side passes heat on, and how much it takes to warm it, decide how
// much passes, so that thin air barely changes the walls round it and a vacuum doesn't at all. Bodies give their heat
// off into the room or suit they're in, and suits pass it on to the air around them. The amounts crossing are worked
// out from the temperatures as they were before any of them, so the order they're gone through in doesn't matter.
func (cc *ChunkContainer) stepHeat(x *ecs.ECSContainer) {
	solids := map[utils.Vec2i]*thermalDef{}
	for _, id := range slices.Sorted(maps.Keys(cc.rooms)) {
		cc.roomTiles(id, func(t utils.Vec2i) bool {
			for _, n := range edgeNeighbours(t) {
				if def, ok := cc.solidThermal(n); ok {
					solids[n] = def
					for _, behind := range edgeNeighbours(n) {
						if def, ok := cc.solidThermal(behind); ok {
							solids[behind] = def
						}
					}
				}
			}
			return true
		})
	}
	cc.keepSolidHeat(solids)

	perTile := map[uint64]climate.Air{}
	roomAir := func(id uint64) climate.Air {
		if a, ok := perTile[id]; ok {
			return a
		}
		r := cc.rooms[id]
		var a climate.Air
		for g, mg := range r.air {
			a[g] = mg / int64(r.tiles)
		}
		perTile[id] = a
		return a
	}
	capacity := map[uint64]int64{}
	roomCapacity := func(id uint64) int64 {
		if c, ok := capacity[id]; ok {
			return c
		}
		capacity[id] = cc.airCapacity(cc.rooms[id].air)
		return capacity[id]
	}
	gained := map[heatNode]int64{}
	tiles := slices.SortedFunc(maps.Keys(solids), func(a, b utils.Vec2i) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })
	for _, t := range tiles {
		def, h := solids[t], cc.solidHeatAt(t)
		self := heatNode{solid: t}
		for _, n := range edgeNeighbours(t) {
			if other, ok := solids[n]; ok {
				if cmp.Or(cmp.Compare(t.Y, n.Y), cmp.Compare(t.X, n.X)) < 0 {
					mJ := heatFlow(edgeConductance(def.conductance, other.conductance), h.temperature, cc.solidHeatAt(n).temperature, def.capacity(), other.capacity())
					gained[self] -= mJ
					gained[heatNode{solid: n}] += mJ
				}
				continue
			}
			if _, ok := cc.solidThermal(n); ok {
				continue
			}
			id, ok := cc.roomOf(n)
			if !ok {
				continue
			}
			if id == 0 {
				outdoors := cc.AmbientTemperature(n, cc.tick)
				gained[self] -= heatFlow(edgeConductance(def.conductance, cc.airConductance(cc.planet.Atmosphere)), h.temperature, outdoors, def.capacity(), 0)
				continue
			}
			r := cc.rooms[id]
			mJ := heatFlow(edgeConductance(def.conductance, cc.airConductance(roomAir(id))), h.temperature, r.temperature, def.capacity(), roomCapacity(id))
			gained[self] -= mJ
			gained[heatNode{room: id}] += mJ
		}
	}

	stepMs := int64(data.AirStepTicks * 1000 / data.TPS * data.HeatTimeScale)
	for _, e := range slices.Sorted(maps.Keys(maps.Collect(ecs.Query1(&x.Bodies, ecs.Without(&x.Disabled))))) {
		pos, err := x.WorldPos(e)
		if err != nil {
			continue
		}
		id, inRoom := cc.roomOf(x.AirTile(pos))
		if sealed, ok := x.WornSealedAir(e); ok {
			suitCapacity := cc.airCapacity(sealed.Contents())
			outsideAir, outside, outsideCapacity := cc.planet.Atmosphere, cc.AmbientTemperature(x.AirTile(pos), cc.tick), int64(0)
			if inRoom && id != 0 {
				outsideAir, outside, outsideCapacity = roomAir(id), cc.rooms[id].temperature, roomCapacity(id)
			}
			lost := heatFlow(data.SealedAirSides*edgeConductance(sealed.Conductance(), cc.airConductance(outsideAir)), sealed.Temperature(), outside, suitCapacity, outsideCapacity)
			sealed.Heat(x.HeatWatts(e)*stepMs-lost, suitCapacity)
			if inRoom && id != 0 {
				gained[heatNode{room: id}] += lost
			}
			continue
		}
		if inRoom && id != 0 {
			gained[heatNode{room: id}] += x.HeatWatts(e) * stepMs
		}
	}

	for _, n := range slices.SortedFunc(maps.Keys(gained), func(a, b heatNode) int {
		return cmp.Or(cmp.Compare(a.room, b.room), cmp.Compare(a.solid.Y, b.solid.Y), cmp.Compare(a.solid.X, b.solid.X))
	}) {
		if n.room != 0 {
			r := cc.rooms[n.room]
			h := heat{temperature: r.temperature, remainder: r.heatRemainder}
			h.take(gained[n], roomCapacity(n.room))
			r.temperature, r.heatRemainder = h.temperature, h.remainder
			continue
		}
		cc.solidHeatAt(n.solid).take(gained[n], solids[n.solid].capacity())
	}
}

// Walls and doors that heat is worked out for start at the temperature of the outdoors there, and those it no longer
// is lose what they had. Heat doesn't make a chunk need saving, so that a cave nothing's been done to isn't saved for the
// warmth of its rock, which starts over at the outdoors' once it's loaded again.
func (cc *ChunkContainer) keepSolidHeat(solids map[utils.Vec2i]*thermalDef) {
	for _, c := range cc.chunks {
		for t := range c.solidHeat {
			if _, ok := solids[t]; !ok {
				delete(c.solidHeat, t)
			}
		}
	}
	for t := range solids {
		c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
		if _, ok := c.solidHeat[t]; !ok {
			c.solidHeat[t] = &heat{temperature: cc.AmbientTemperature(t, cc.tick)}
		}
	}
}

func (cc *ChunkContainer) solidHeatAt(t utils.Vec2i) *heat {
	return cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}].solidHeat[t]
}

// What heat is worked out for on the tile: a wall or door's, or the air's, false where there's neither, or a vacuum,
// which has no temperature
func (cc *ChunkContainer) TemperatureAt(t utils.Vec2i) (climate.Centicelsius, bool) {
	if c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]; ok {
		if h, ok := c.solidHeat[t]; ok {
			return h.temperature, true
		}
	}
	air, ok := cc.AirAt(t)
	if !ok || air.Total() == 0 {
		return 0, false
	}
	return cc.AirTemperature(t), true
}
