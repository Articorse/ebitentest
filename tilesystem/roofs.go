package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/utils"
	"fmt"
	"slices"
)

// How far a colonist has got with a roof, which carries over to whoever works on it next
type roofWork struct {
	// Toward a built roof, rather than taking one down
	building bool
	work     int
}

type roofWorkDto struct {
	X, Y     int
	Building bool
	Work     int
}

// Whether there's a roof to build over the tile, on the roof area where there's none and something holds one up, or a
// built one to take down, on the no-roof area
func (cc *ChunkContainer) RoofJob(t utils.Vec2i) bool {
	roof, ok := cc.RoofAt(t)
	switch {
	case !ok:
		return false
	case cc.InArea(data.Area_Roof, t):
		return roof == data.Roof_None && cc.roofSupported(t)
	case cc.InArea(data.Area_NoRoof, t):
		return roof == data.Roof_Built
	}
	return false
}

// The tiles from first to last with a roof to build or take down, row by row
func (cc *ChunkContainer) RoofJobsIn(first, last utils.Vec2i) []utils.Vec2i {
	jobs := append(cc.AreaTiles(data.Area_Roof, first, last), cc.AreaTiles(data.Area_NoRoof, first, last)...)
	jobs = slices.DeleteFunc(jobs, func(t utils.Vec2i) bool { return !cc.RoofJob(t) })
	slices.SortFunc(jobs, func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return jobs
}

// Works on the roof job over the tile, finishing it once it's had data.RoofWork. Worked is false without a job there.
func (cc *ChunkContainer) WorkOnRoof(t utils.Vec2i, work int) (worked, done bool) {
	if !cc.RoofJob(t) {
		return false, false
	}
	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	building := cc.InArea(data.Area_Roof, t)
	w := c.roofWork[t]
	if w.building != building {
		w = roofWork{building: building}
	}
	w.work += work
	c.terrainModified = true
	if w.work < data.RoofWork {
		c.roofWork[t] = w
		return true, false
	}
	if building {
		cc.setRoof(c, t, data.Roof_Built)
		return true, true
	}
	cc.setRoof(c, t, data.Roof_None)
	cc.collapseAround(t)
	return true, true
}

func (cc *ChunkContainer) setRoof(c *chunk, t utils.Vec2i, roof data.RoofEnum) {
	c.roofs[c.tileIndex(t)] = roof
	delete(c.roofWork, t)
	c.terrainModified = true
	cc.sightVersion++
}

// Over a wall or door, or joined edge to edge through roofs, within data.RoofSupportTiles steps, to a roof over one, to a
// rock roof, or to somewhere that isn't loaded, which may hold it up
func (cc *ChunkContainer) roofSupported(t utils.Vec2i) bool {
	steps := map[utils.Vec2i]int{t: 0}
	queue := []utils.Vec2i{t}
	for i := 0; i < len(queue); i++ {
		at := queue[i]
		if holds, loaded := cc.holdsRoofUp(at); holds || !loaded {
			return true
		}
		if steps[at] == data.RoofSupportTiles {
			continue
		}
		for _, n := range edgeNeighbours(at) {
			if _, seen := steps[n]; seen {
				continue
			}
			if roof, ok := cc.RoofAt(n); ok && roof == data.Roof_None {
				continue
			}
			steps[n] = steps[at] + 1
			queue = append(queue, n)
		}
	}
	return false
}

// A wall or door, or a rock roof, which rock holds up
func (cc *ChunkContainer) holdsRoofUp(t utils.Vec2i) (holds, loaded bool) {
	tiles, ok := cc.TileAt(t)
	if !ok {
		return false, false
	}
	if cc.isWall(tiles[data.TileLayer_Wall]) {
		return true, true
	}
	if _, door := cc.DoorAt(t); door {
		return true, true
	}
	roof, _ := cc.RoofAt(t)
	return roof == data.Roof_Rock, true
}

// Built roofs that what was taken off or out from under the tile held up, and nothing else does, fall in. Only roofs
// within data.RoofSupportTiles steps of it can have been held up through it, and one that falls held up none of the
// others, since they'd have held it up in turn.
func (cc *ChunkContainer) collapseAround(t utils.Vec2i) {
	r := data.RoofSupportTiles
	for dy := -r; dy <= r; dy++ {
		for dx := abs(dy) - r; dx <= r-abs(dy); dx++ {
			n := utils.Vec2i{X: t.X + dx, Y: t.Y + dy}
			if roof, ok := cc.RoofAt(n); ok && roof == data.Roof_Built && !cc.roofSupported(n) {
				cc.setRoof(cc.chunks[utils.Vec2i{X: floorDiv(n.X, data.ChunkSize), Y: floorDiv(n.Y, data.ChunkSize)}], n, data.Roof_None)
			}
		}
	}
}

var roofNames = map[data.RoofEnum]string{data.Roof_None: "None", data.Roof_Rock: "Rock", data.Roof_Built: "Built"}

// The roof over the tile and how far work on it has got, for the debug tooltip
func (cc *ChunkContainer) RoofDebugLines(t utils.Vec2i) []string {
	roof, ok := cc.RoofAt(t)
	if !ok {
		return nil
	}
	lines := []string{"Roof " + roofNames[roof]}
	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if w, ok := c.roofWork[t]; ok {
		verb := "Taking down"
		if w.building {
			verb = "Building"
		}
		lines = append(lines, fmt.Sprintf("%s %d/%d", verb, w.work, data.RoofWork))
	}
	return lines
}

// In tile order, so that saving the same chunk twice writes the same file
func roofWorkDtos(c *chunk) []roofWorkDto {
	dtos := make([]roofWorkDto, 0, len(c.roofWork))
	for t, w := range c.roofWork {
		dtos = append(dtos, roofWorkDto{X: t.X, Y: t.Y, Building: w.building, Work: w.work})
	}
	slices.SortFunc(dtos, func(a, b roofWorkDto) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return dtos
}

func roofWorkFromDtos(dtos []roofWorkDto) map[utils.Vec2i]roofWork {
	work := make(map[utils.Vec2i]roofWork, len(dtos))
	for _, d := range dtos {
		work[utils.Vec2i{X: d.X, Y: d.Y}] = roofWork{building: d.Building, work: d.Work}
	}
	return work
}
