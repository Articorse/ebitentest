package tilesystem

import (
	"ebittest/data"
	"ebittest/utils"
	"fmt"
	"maps"
	"slices"
)

// The walkable tiles of a chunk that are connected without going through a door or out of the chunk. A door's tile is a
// region of its own, so that whether it can be gone through is up to whoever's walking.
type chunkRegion struct {
	tiles int
	door  bool
	// The regions next to the door
	links []int32
}

// The regions of every active chunk, joined up across chunk edges. Ids follow the chunks in order, so that the same
// world gives the same ids.
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

func (cc *ChunkContainer) chunkRegions(c *chunk) {
	if c.regionOf != nil {
		return
	}
	if c.walkSpeeds == nil {
		c.walkSpeeds = cc.chunkWalkSpeeds(c)
	}
	var door [data.ChunkSize * data.ChunkSize]bool
	for t, f := range c.features {
		door[c.tileIndex(t)] = cc.featureAtlas[f].door != nil
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
		if regionOf[start] != -1 || c.walkSpeeds[start] == 0 {
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
				if regionOf[j] == -1 && c.walkSpeeds[j] > 0 && !door[j] {
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
	c.regionOf, c.regions = regionOf, regions
}

func (cc *ChunkContainer) regions() *regionGraph {
	if cc.regionGraph != nil {
		return cc.regionGraph
	}
	g := &regionGraph{first: map[utils.Vec2i]int32{}, groups: map[data.FactionEnum]*regionGroups{}}
	positions := slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos)
	for _, pos := range positions {
		c := cc.chunks[pos]
		cc.chunkRegions(c)
		g.first[pos] = int32(len(g.nodes))
		for _, r := range c.regions {
			n := regionNode{tiles: r.tiles, door: r.door}
			for _, l := range r.links {
				n.links = append(n.links, g.first[pos]+l)
			}
			g.nodes = append(g.nodes, n)
		}
	}
	for _, pos := range positions {
		c := cc.chunks[pos]
		for t, f := range c.features {
			if local := c.regionOf[c.tileIndex(t)]; local != -1 && cc.featureAtlas[f].door != nil {
				g.nodes[g.first[pos]+local].faction = c.doorFactions[t]
			}
		}
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		for _, i := range chunkEdgeIndices {
			local := c.regionOf[i]
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
				if other := nc.regionOf[nc.tileIndex(n)]; other != -1 {
					g.nodes[id].links = append(g.nodes[id].links, g.first[npos]+other)
				}
			}
		}
	}
	cc.regionGraph = g
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

// -1 where the tile isn't walked on or isn't active
func (cc *ChunkContainer) regionAt(t utils.Vec2i) int32 {
	pos := utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}
	c, ok := cc.chunks[pos]
	if !ok {
		return -1
	}
	g := cc.regions()
	local := c.regionOf[c.tileIndex(t)]
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

// Whether the faction can walk from one tile to the other, going through only its own doors. True where either isn't
// walked on, which is left to paths to find out, such as for a target standing half in a wall.
func (cc *ChunkContainer) Reachable(from, to utils.Vec2i, faction data.FactionEnum) bool {
	a, b := cc.regionAt(from), cc.regionAt(to)
	if a == -1 || b == -1 {
		return true
	}
	g := cc.regions()
	for _, x := range g.groupsFrom(a, faction) {
		if slices.Contains(g.groupsFrom(b, faction), x) {
			return true
		}
	}
	return false
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
	id := cc.regionAt(t)
	if id == -1 {
		return []string{"No region"}
	}
	g := cc.regions()
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
