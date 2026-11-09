package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"
)

// The walkable tiles of a chunk that are connected without going through a door or out of the chunk. A door's tile is a
// region of its own, so that whether it can be gone through is up to whoever's walking. For a body bigger than a tile,
// the tiles walkable are those it fits in the middle of, and a tile where it would be over a door isn't, since doors
// are a tile wide.
type chunkRegion struct {
	tiles int
	door  bool
	// The regions next to the door
	links []int32
}

// A chunk's regions for one size of body, and which of them each tile is in, -1 where it isn't walked on
type chunkRegionSet struct {
	regionOf *[data.ChunkSize * data.ChunkSize]int32
	regions  []chunkRegion
}

// The regions of every active chunk for one size of body, joined up across chunk edges. Ids follow the chunks in order,
// so that the same world gives the same ids.
type regionGraph struct {
	first map[utils.Vec2i]int32
	nodes []regionNode
	// For each faction that's asked about, the group of regions it can walk between that each region is in
	groups map[data.FactionEnum]*regionGroups
}

type regionNode struct {
	tiles int
	// Whose door it is, for a door's region
	door    bool
	faction data.FactionEnum
	// Next to a chunk that isn't active, which leaves how far it goes unknown
	open  bool
	links []int32
}

type regionGroups struct {
	of    []int32
	tiles []int
	open  []bool
}

// Anyone can stand on a door, but only its own faction can go through it
func (n regionNode) passableFor(faction data.FactionEnum) bool {
	return !n.door || (n.faction != data.Faction_None && n.faction == faction)
}

// A tile walked on, where a door is walked on too, whoever's it is
func (cc *ChunkContainer) regionWalkable(c *chunk, t utils.Vec2i) bool {
	if c.walkSpeeds == nil {
		c.walkSpeeds = cc.chunkWalkSpeeds(c)
	}
	return c.walkSpeeds[c.tileIndex(t)] > 0
}

// The tiles along a chunk's edges, by their index in it, each once
var chunkEdgeIndices = func() []int {
	indices := []int{}
	for i := range data.ChunkSize * data.ChunkSize {
		if x, y := i%data.ChunkSize, i/data.ChunkSize; x == 0 || x == data.ChunkSize-1 || y == 0 || y == data.ChunkSize-1 {
			indices = append(indices, i)
		}
	}
	return indices
}()

func edgeNeighbours(t utils.Vec2i) [4]utils.Vec2i {
	return [4]utils.Vec2i{{X: t.X - 1, Y: t.Y}, {X: t.X + 1, Y: t.Y}, {X: t.X, Y: t.Y - 1}, {X: t.X, Y: t.Y + 1}}
}

func (cc *ChunkContainer) chunkRegions(pos utils.Vec2i, c *chunk, box bodyBox) *chunkRegionSet {
	if set, ok := c.regionSets[box]; ok {
		return set
	}
	if c.walkSpeeds == nil {
		c.walkSpeeds = cc.chunkWalkSpeeds(c)
	}
	var walkable, door [data.ChunkSize * data.ChunkSize]bool
	if box == (bodyBox{}) {
		for i, speed := range c.walkSpeeds {
			walkable[i] = speed > 0
		}
		for t, f := range c.features {
			door[c.tileIndex(t)] = cc.featureAtlas[f].door != nil
		}
	} else {
		speeds := cc.BodySpeeds(box.lo, box.hi)
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		for i := range walkable {
			speed, overDoor := speeds(utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize})
			walkable[i] = speed > 0 && !overDoor
		}
	}
	regionOf := &[data.ChunkSize * data.ChunkSize]int32{}
	for i := range regionOf {
		regionOf[i] = -1
	}
	regions := []chunkRegion{}
	neighbours := func(i int, visit func(j int)) {
		x, y := i%data.ChunkSize, i/data.ChunkSize
		if x > 0 {
			visit(i - 1)
		}
		if x < data.ChunkSize-1 {
			visit(i + 1)
		}
		if y > 0 {
			visit(i - data.ChunkSize)
		}
		if y < data.ChunkSize-1 {
			visit(i + data.ChunkSize)
		}
	}

	queue := []int{}
	for start := range regionOf {
		if regionOf[start] != -1 || !walkable[start] {
			continue
		}
		id := int32(len(regions))
		regionOf[start] = id
		if door[start] {
			regions = append(regions, chunkRegion{tiles: 1, door: true})
			continue
		}
		region := chunkRegion{}
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			region.tiles++
			neighbours(i, func(j int) {
				if regionOf[j] == -1 && walkable[j] && !door[j] {
					regionOf[j] = id
					queue = append(queue, j)
				}
			})
		}
		regions = append(regions, region)
	}

	for i, id := range regionOf {
		if id == -1 || !regions[id].door {
			continue
		}
		neighbours(i, func(j int) {
			if other := regionOf[j]; other != -1 {
				regions[id].links = append(regions[id].links, other)
				regions[other].links = append(regions[other].links, id)
			}
		})
	}
	set := &chunkRegionSet{regionOf: regionOf, regions: regions}
	if c.regionSets == nil {
		c.regionSets = map[bodyBox]*chunkRegionSet{}
	}
	c.regionSets[box] = set
	return set
}

func (cc *ChunkContainer) regions(box bodyBox) *regionGraph {
	if g, ok := cc.regionGraphs[box]; ok {
		return g
	}
	g := &regionGraph{first: map[utils.Vec2i]int32{}, groups: map[data.FactionEnum]*regionGroups{}}
	positions := slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos)
	sets := map[utils.Vec2i]*chunkRegionSet{}
	for _, pos := range positions {
		set := cc.chunkRegions(pos, cc.chunks[pos], box)
		sets[pos] = set
		g.first[pos] = int32(len(g.nodes))
		for _, r := range set.regions {
			n := regionNode{tiles: r.tiles, door: r.door}
			for _, l := range r.links {
				n.links = append(n.links, g.first[pos]+l)
			}
			g.nodes = append(g.nodes, n)
		}
	}
	for _, pos := range positions {
		c, set := cc.chunks[pos], sets[pos]
		for t, f := range c.features {
			if local := set.regionOf[c.tileIndex(t)]; local != -1 && cc.featureAtlas[f].door != nil {
				g.nodes[g.first[pos]+local].faction = c.doorFactions[t]
			}
		}
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		for _, i := range chunkEdgeIndices {
			local := set.regionOf[i]
			if local == -1 {
				continue
			}
			id := g.first[pos] + local
			t := utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize}
			for _, n := range edgeNeighbours(t) {
				npos := utils.Vec2i{X: floorDiv(n.X, data.ChunkSize), Y: floorDiv(n.Y, data.ChunkSize)}
				if npos == pos {
					continue
				}
				nc, ok := cc.chunks[npos]
				if !ok {
					g.nodes[id].open = true
					continue
				}
				if other := sets[npos].regionOf[nc.tileIndex(n)]; other != -1 {
					g.nodes[id].links = append(g.nodes[id].links, g.first[npos]+other)
				}
			}
		}
	}
	if cc.regionGraphs == nil {
		cc.regionGraphs = map[bodyBox]*regionGraph{}
	}
	cc.regionGraphs[box] = g
	return g
}

func (g *regionGraph) groupsFor(faction data.FactionEnum) *regionGroups {
	if groups, ok := g.groups[faction]; ok {
		return groups
	}
	groups := &regionGroups{of: make([]int32, len(g.nodes))}
	for i := range groups.of {
		groups.of[i] = -1
	}
	queue := []int32{}
	for start := range g.nodes {
		if groups.of[start] != -1 {
			continue
		}
		group := int32(len(groups.tiles))
		groups.tiles, groups.open = append(groups.tiles, 0), append(groups.open, false)
		groups.of[start] = group
		queue = append(queue[:0], int32(start))
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			n := g.nodes[id]
			groups.tiles[group] += n.tiles
			groups.open[group] = groups.open[group] || n.open
			if !n.passableFor(faction) {
				continue
			}
			for _, l := range n.links {
				if groups.of[l] == -1 && g.nodes[l].passableFor(faction) {
					groups.of[l] = group
					queue = append(queue, l)
				}
			}
		}
	}
	g.groups[faction] = groups
	return groups
}

// -1 where the tile isn't walked on by the body or isn't active
func (cc *ChunkContainer) regionAt(t utils.Vec2i, box bodyBox) int32 {
	pos := utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}
	c, ok := cc.chunks[pos]
	if !ok {
		return -1
	}
	g := cc.regions(box)
	local := c.regionSets[box].regionOf[c.tileIndex(t)]
	if local == -1 {
		return -1
	}
	return g.first[pos] + local
}

// The groups a walker on the region can get to: its own, or those on either side of a door it can't go through
func (g *regionGraph) groupsFrom(id int32, faction data.FactionEnum) []int32 {
	groups := g.groupsFor(faction)
	if g.nodes[id].passableFor(faction) {
		return []int32{groups.of[id]}
	}
	from := []int32{}
	for _, l := range g.nodes[id].links {
		if g.nodes[l].passableFor(faction) {
			from = append(from, groups.of[l])
		}
	}
	return from
}

// Whether a body of the faction reaching lo to hi tiles round its own can walk from one tile to touching the other,
// going through only its faction's doors. For a body of one tile, that's walking onto it, and true where it isn't
// walked on, which is left to paths to find out, such as for a target standing half in a wall. A bigger body touches
// it from any tile it fits in the middle of whose tiles reach next to it. True where the body doesn't fit on the tile
// it's walking from, which is also left to paths.
func (cc *ChunkContainer) Reachable(from, to utils.Vec2i, faction data.FactionEnum, lo, hi utils.Vec2i) bool {
	box := bodyBox{lo, hi}
	a := cc.regionAt(from, box)
	if a == -1 {
		return true
	}
	g := cc.regions(box)
	groups := g.groupsFrom(a, faction)
	if box == (bodyBox{}) {
		b := cc.regionAt(to, box)
		return b == -1 || shareGroup(g.groupsFrom(b, faction), groups)
	}
	for y := to.Y - hi.Y - 1; y <= to.Y-lo.Y+1; y++ {
		for x := to.X - hi.X - 1; x <= to.X-lo.X+1; x++ {
			if cc.inGroups(utils.Vec2i{X: x, Y: y}, box, faction, groups) {
				return true
			}
		}
	}
	return false
}

func (cc *ChunkContainer) inGroups(t utils.Vec2i, box bodyBox, faction data.FactionEnum, groups []int32) bool {
	id := cc.regionAt(t, box)
	if id == -1 {
		return false
	}
	return shareGroup(cc.regions(box).groupsFrom(id, faction), groups)
}

func shareGroup(a, b []int32) bool {
	return slices.ContainsFunc(a, func(x int32) bool { return slices.Contains(b, x) })
}

// The tile closest to toward, within radius tiles of it, that a body of the faction reaching lo to hi round its own can
// walk to from the tile, the first in rows of those as close. False when there's none, or the body doesn't fit where
// it's walking from.
func (cc *ChunkContainer) ClosestReachable(
	from, toward utils.Vec2i,
	faction data.FactionEnum,
	lo, hi utils.Vec2i,
	radius int,
) (utils.Vec2i, bool) {
	box := bodyBox{lo, hi}
	a := cc.regionAt(from, box)
	if a == -1 {
		return utils.Vec2i{}, false
	}
	groups := cc.regions(box).groupsFrom(a, faction)
	best, bestDist, found := utils.Vec2i{}, 0, false
	for r := 0; r <= radius; r++ {
		if found && r*r > bestDist {
			break
		}
		for y := toward.Y - r; y <= toward.Y+r; y++ {
			for x := toward.X - r; x <= toward.X+r; x++ {
				t := utils.Vec2i{X: x, Y: y}
				if max(utils.AbsInt(x-toward.X), utils.AbsInt(y-toward.Y)) != r {
					continue
				}
				if d := utils.Dist2(t, toward); (!found || d < bestDist) && cc.inGroups(t, box, faction, groups) {
					best, bestDist, found = t, d, true
				}
			}
		}
	}
	return best, found
}

// The tiles the faction can walk to from the tile, as if the closed one couldn't be walked on, up to limit of them.
// Enclosed if they're all there are, with none next to a chunk that isn't active, whose size isn't known.
func (cc *ChunkContainer) Enclosure(
	from, closed utils.Vec2i,
	faction data.FactionEnum,
	limit int,
) (tiles []utils.Vec2i, enclosed bool) {
	walkable := func(t utils.Vec2i) (ok, active bool) {
		c, active := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
		if !active {
			return false, false
		}
		if !cc.regionWalkable(c, t) {
			return false, true
		}
		if door, ok := cc.DoorAt(t); ok && (door.Faction == data.Faction_None || door.Faction != faction) {
			return false, true
		}
		return true, true
	}
	if ok, _ := walkable(from); !ok || from == closed {
		return nil, false
	}
	seen := map[utils.Vec2i]bool{from: true}
	tiles = []utils.Vec2i{from}
	for i := 0; i < len(tiles); i++ {
		t := tiles[i]
		for _, n := range edgeNeighbours(t) {
			if seen[n] || n == closed {
				continue
			}
			ok, active := walkable(n)
			if !active {
				return tiles, false
			}
			if !ok {
				continue
			}
			if len(tiles) == limit {
				return tiles, false
			}
			seen[n] = true
			tiles = append(tiles, n)
		}
	}
	return tiles, true
}

var factionNames = map[data.FactionEnum]string{data.Faction_Colony: "Colony", data.Faction_Monsters: "Monsters"}

// The tile's region and the groups the colony and monsters can walk between that it's in, for the debug tooltip
func (cc *ChunkContainer) RegionDebugLines(t utils.Vec2i) []string {
	lines := cc.regionDebugLines(t, bodyBox{})
	boxes := slices.Collect(maps.Keys(cc.regionGraphs))
	slices.SortFunc(boxes, func(a, b bodyBox) int {
		return cmp.Or(cmp.Compare(a.hi.X-a.lo.X, b.hi.X-b.lo.X), cmp.Compare(a.hi.Y-a.lo.Y, b.hi.Y-b.lo.Y))
	})
	for _, box := range boxes {
		if box == (bodyBox{}) {
			continue
		}
		size := fmt.Sprintf("%dx%d ", box.hi.X-box.lo.X+1, box.hi.Y-box.lo.Y+1)
		for _, line := range cc.regionDebugLines(t, box) {
			lines = append(lines, size+line)
		}
	}
	return lines
}

func (cc *ChunkContainer) regionDebugLines(t utils.Vec2i, box bodyBox) []string {
	id := cc.regionAt(t, box)
	if id == -1 {
		return []string{"No region"}
	}
	g := cc.regions(box)
	n := g.nodes[id]
	lines := []string{fmt.Sprintf("Region %d: %d tiles", id, n.tiles)}
	if n.door {
		lines[0] += fmt.Sprintf(", door of faction %d", n.faction)
	}
	for _, faction := range []data.FactionEnum{data.Faction_Colony, data.Faction_Monsters} {
		groups := g.groupsFor(faction)
		for _, group := range g.groupsFrom(id, faction) {
			line := fmt.Sprintf("%s group %d: %d tiles", factionNames[faction], group, groups.tiles[group])
			if groups.open[group] {
				line += ", open"
			}
			lines = append(lines, line)
		}
	}
	return lines
}
