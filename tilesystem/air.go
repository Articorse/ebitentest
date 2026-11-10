package tilesystem

import (
	"cmp"
	"ebittest/climate"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"
)

// The air of a room as it's worked out again: from each room it was part of, its share of that room's air by the tiles
// it took over, and from the outdoors, theirs for the rest. A room built within another so has that one's air.
func (cc *ChunkContainer) airFrom(before map[uint64]*room, was map[uint64]int, tiles int, first utils.Vec2i) *room {
	r := &room{tiles: tiles}
	outdoors := tiles
	heat := int64(0)
	for _, id := range slices.Sorted(maps.Keys(was)) {
		old, ok := before[id]
		if !ok || old.unknownAir || old.tiles == 0 {
			continue
		}
		overlap := was[id]
		outdoors -= overlap
		for g, mg := range old.air {
			share := mg * int64(overlap) / int64(old.tiles)
			r.air[g] += share
			heat += share * int64(old.temperature)
		}
		if old.air.Total() == 0 {
			r.temperature = old.temperature
		}
	}
	ambient := cc.AmbientTemperature(first, cc.tick)
	for g, mg := range cc.planet.Atmosphere {
		share := mg * int64(outdoors)
		r.air[g] += share
		heat += share * int64(ambient)
	}
	switch total := r.air.Total(); {
	case total > 0:
		r.temperature = climate.Centicelsius(heat / total)
	case outdoors > 0:
		r.temperature = ambient
	}
	return r
}

// Air flows across it, from the side with more of a gas per tile to the side with less, gas by gas. Rooms are numbered by
// their ids; 0 is the outdoors.
type opening struct {
	a, b uint64
	// Openings alike, such as a room's unroofed tiles, go together
	count int64
	// Of the outdoors' air coming in through it
	ambient climate.Centicelsius
}

// Lets air through the openings of the rooms that are active: open doors, and tiles open to the sky. The amounts
// crossing are worked out from the air as it was before any of them, so the order they're gone through in doesn't
// matter.
func (cc *ChunkContainer) stepAir() {
	cc.updateRooms()
	openings := cc.openings()
	count := map[uint64]int64{}
	for _, o := range openings {
		count[o.a] += o.count
		count[o.b] += o.count
	}
	perTile := func(id uint64) climate.Air {
		if id == 0 {
			return cc.planet.Atmosphere
		}
		r := cc.rooms[id]
		var a climate.Air
		for g, mg := range r.air {
			a[g] = mg / int64(r.tiles)
		}
		return a
	}
	// So that a small room with many openings isn't emptied past its neighbours' air in one step: no more than
	// half the difference crosses all of a room's openings together
	rate := func(o opening) (num, den int64) {
		num, den = data.OpeningFlowNum*o.count, data.OpeningFlowDen
		for _, id := range []uint64{o.a, o.b} {
			if id == 0 {
				continue
			}
			if limit := int64(cc.rooms[id].tiles) * o.count; num*2*count[id] > limit*den {
				num, den = limit, 2*count[id]
			}
		}
		return num, den
	}

	type inflow struct {
		mg          int64
		temperature climate.Centicelsius
	}
	change := map[uint64]*climate.Air{}
	outflow := map[uint64]int64{}
	inflows := map[uint64][]inflow{}
	for _, o := range openings {
		num, den := rate(o)
		from, to := perTile(o.a), perTile(o.b)
		for g := range from {
			mg := (from[g] - to[g]) * num / den
			src, dst := o.a, o.b
			if mg < 0 {
				mg, src, dst = -mg, o.b, o.a
			}
			if mg == 0 {
				continue
			}
			temperature := o.ambient
			if src != 0 {
				temperature = cc.rooms[src].temperature
				if change[src] == nil {
					change[src] = &climate.Air{}
				}
				change[src][g] -= mg
				outflow[src] += mg
			}
			if dst != 0 {
				if change[dst] == nil {
					change[dst] = &climate.Air{}
				}
				change[dst][g] += mg
				inflows[dst] = append(inflows[dst], inflow{mg: mg, temperature: temperature})
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(change)) {
		r := cc.rooms[id]
		staying := max(0, r.air.Total()-outflow[id])
		mass, heat := staying, staying*int64(r.temperature)
		for _, in := range inflows[id] {
			mass += in.mg
			heat += in.mg * int64(in.temperature)
		}
		if mass > 0 {
			r.temperature = climate.Centicelsius(heat / mass)
		}
		for g, mg := range change[id] {
			r.air[g] = max(0, r.air[g]+mg)
		}
	}
}

// Of the rooms with an active tile, in a set order
func (cc *ChunkContainer) openings() []opening {
	unroofed := map[uint64]int64{}
	ambient := map[uint64]climate.Centicelsius{}
	doors := map[[2]uint64]*opening{}
	for _, pos := range slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos) {
		c := cc.chunks[pos]
		p := cc.piecesOf(c)
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		inRooms := slices.ContainsFunc(c.pieceRooms, func(id uint64) bool { return id != 0 })
		for i, local := range p.pieceOf {
			if !inRooms {
				break
			}
			if local == -1 || c.pieceRooms[local] == 0 {
				continue
			}
			id := c.pieceRooms[local]
			t := utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize}
			if c.roofs[i] == data.Roof_None {
				if unroofed[id] == 0 {
					ambient[id] = cc.AmbientTemperature(t, cc.tick)
				}
				unroofed[id]++
			}
		}
		// Only doors of a faction are kept there, which are the only ones that open, and far fewer than the features
		for _, t := range slices.SortedFunc(maps.Keys(c.doorFactions), compareTiles) {
			door, ok := cc.DoorAt(t)
			if !ok || !door.Open {
				continue
			}
			sides := []uint64{}
			for _, n := range [4]utils.Vec2i{{X: t.X - 1, Y: t.Y}, {X: t.X + 1, Y: t.Y}, {X: t.X, Y: t.Y - 1}, {X: t.X, Y: t.Y + 1}} {
				if _, active := cc.TileAt(n); !active || cc.isAirTight(n) {
					continue
				}
				id, _ := cc.roomOf(n)
				if !slices.Contains(sides, id) {
					sides = append(sides, id)
				}
			}
			slices.Sort(sides)
			for i, a := range sides {
				for _, b := range sides[i+1:] {
					if o, ok := doors[[2]uint64{a, b}]; ok {
						o.count++
					} else {
						doors[[2]uint64{a, b}] = &opening{a: a, b: b, count: 1, ambient: cc.AmbientTemperature(t, cc.tick)}
					}
				}
			}
		}
	}
	openings := []opening{}
	for _, id := range slices.Sorted(maps.Keys(unroofed)) {
		openings = append(openings, opening{a: 0, b: id, count: unroofed[id], ambient: ambient[id]})
	}
	for _, key := range slices.SortedFunc(maps.Keys(doors), func(a, b [2]uint64) int {
		return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1]))
	}) {
		openings = append(openings, *doors[key])
	}
	return openings
}

func compareTiles(a, b utils.Vec2i) int {
	return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
}

// The room the tile's in, without working the rooms out again first, 0 outdoors
func (cc *ChunkContainer) roomOf(t utils.Vec2i) (uint64, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok || c.pieceRooms == nil {
		return 0, false
	}
	local := cc.piecesOf(c).pieceOf[c.tileIndex(t)]
	if local == -1 {
		return 0, false
	}
	return c.pieceRooms[local], true
}

// The air on the tile, per tile, whether in a room or outdoors. False on a wall or door, or where nothing is active.
func (cc *ChunkContainer) AirAt(t utils.Vec2i) (climate.Air, bool) {
	cc.updateRooms()
	if _, active := cc.TileAt(t); !active || cc.isAirTight(t) {
		return climate.Air{}, false
	}
	id, _ := cc.roomOf(t)
	if id == 0 {
		return cc.planet.Atmosphere, true
	}
	r := cc.rooms[id]
	var a climate.Air
	for g, mg := range r.air {
		a[g] = mg / int64(r.tiles)
	}
	return a, true
}

// How warm the air on the tile is, in a room or outdoors
func (cc *ChunkContainer) AirTemperature(t utils.Vec2i) climate.Centicelsius {
	if id, _ := cc.roomOf(t); id != 0 {
		return cc.rooms[id].temperature
	}
	return cc.AmbientTemperature(t, cc.tick)
}

// The air on the tile, its temperature and what it's made of, for the debug tooltip
func (cc *ChunkContainer) AirDebugLines(t utils.Vec2i) []string {
	air, ok := cc.AirAt(t)
	if !ok {
		return nil
	}
	temperature := cc.AirTemperature(t)
	total := air.Total()
	if total == 0 {
		return []string{"Vacuum"}
	}
	lines := []string{fmt.Sprintf("Air %.1f °C, %d g per tile", temperature.Celsius(), roundToGrams(total))}
	for g, mg := range air {
		if gas, ok := cc.gases[data.GasEnum(g)]; ok && mg > 0 {
			lines = append(lines, fmt.Sprintf("  %s %d g (%.0f%%)", gas.Name, roundToGrams(mg), 100*float64(mg)/float64(total)))
		}
	}
	return lines
}

func roundToGrams(mg int64) int64 {
	return (mg + data.MilligramsPerGram/2) / data.MilligramsPerGram
}

// What breathes in a room takes a step's worth of oxygen from its air, or what there is, and gives as much back as
// carbon dioxide. The outdoors' air is the planet's, which nothing changes.
func (cc *ChunkContainer) breathe(x *ecs.ECSContainer) {
	for _, e := range slices.Sorted(maps.Keys(maps.Collect(ecs.Query1(&x.Bodies, ecs.Without(&x.Disabled))))) {
		perSecond := x.OxygenPerSecond(e)
		pos, err := x.WorldPos(e)
		if perSecond == 0 || err != nil {
			continue
		}
		id, ok := cc.roomOf(x.AirTile(pos))
		if !ok || id == 0 {
			continue
		}
		r := cc.rooms[id]
		mg := min(r.air[data.Gas_Oxygen], perSecond*data.AirStepTicks/data.TPS)
		r.air[data.Gas_Oxygen] -= mg
		r.air[data.Gas_CarbonDioxide] += mg
	}
}

// In a room colder than where a gas condenses, some of it drops out of the air as its item, to the milligram, onto the
// room's active tiles row by row where there's room, and what doesn't fit stays in the air. The outdoors' air never
// changes.
func (cc *ChunkContainer) condense(x *ecs.ECSContainer) {
	for _, id := range slices.Sorted(maps.Keys(cc.rooms)) {
		r := cc.rooms[id]
		for _, g := range slices.Sorted(maps.Keys(cc.gases)) {
			gas := cc.gases[g]
			if r.temperature >= gas.Condense {
				continue
			}
			mg := int(max(r.air[g]/data.CondenseFractionDen, min(r.air[g], data.CondenseLeastMg)))
			if mg == 0 {
				continue
			}
			cc.roomTiles(id, func(t utils.Vec2i) bool {
				room := x.GroundRoomFor(t, gas.Item)
				if room <= 0 {
					return true
				}
				on, _ := cc.GroundStack(t)
				n := min(room, mg)
				if cc.SetGroundStack(t, ecs.GroundStack{Kind: gas.Item, Amount: on.Amount + n}) {
					mg -= n
					r.air[g] -= int64(n)
				}
				return mg > 0
			})
		}
	}
}

// Stacks of a gas's item lying where it's warmer than where the gas condenses go back into the air bit by bit: into
// the room they're in, or outdoors, where they're lost to the planet's air
func (cc *ChunkContainer) boil() {
	gasOf := map[data.ItemEnum]climate.Gas{}
	for _, gas := range cc.gases {
		gasOf[gas.Item] = gas
	}
	for _, pos := range slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos) {
		c := cc.chunks[pos]
		for _, t := range slices.SortedFunc(maps.Keys(c.ground), compareTiles) {
			stack := c.ground[t]
			gas, isGas := gasOf[stack.Kind]
			if !isGas || stack.Unique != nil || cc.AirTemperature(t) <= gas.Condense {
				continue
			}
			n := max(stack.Amount/data.BoilFractionDen, min(stack.Amount, data.BoilLeastMg))
			stack.Amount -= n
			cc.SetGroundStack(t, stack)
			if id, _ := cc.roomOf(t); id != 0 {
				cc.rooms[id].air[gas.Id] += int64(n)
			}
		}
	}
}
