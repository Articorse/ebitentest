package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"slices"
)

type Door struct {
	Vertical bool
	Open     bool
	Faction  data.FactionEnum
}

func doorVariant(defs map[data.FeatureEnum]featureDef, kind string, vertical, open bool) (data.FeatureEnum, bool) {
	for id, def := range defs {
		if d := def.door; d != nil && d.kind == kind && d.vertical == vertical && d.open == open {
			return id, true
		}
	}
	return data.Feature_None, false
}

func (cc *ChunkContainer) DoorAt(t utils.Vec2i) (Door, bool) {
	c, def, ok := cc.featureOn(t)
	if !ok || def.door == nil {
		return Door{}, false
	}
	return Door{Vertical: def.door.vertical, Open: def.door.open, Faction: c.doorFactions[t]}, true
}

func (cc *ChunkContainer) ClosedDoor(t utils.Vec2i) (vertical bool, faction data.FactionEnum, ok bool) {
	door, ok := cc.DoorAt(t)
	if !ok || door.Open {
		return false, data.Faction_None, false
	}
	return door.Vertical, door.Faction, true
}

// What a door joins onto: a wall, a door, or a site for either
func (cc *ChunkContainer) joinsDoor(t utils.Vec2i) bool {
	if _, def, ok := cc.featureOn(t); ok && def.door != nil {
		return true
	}
	if _, _, def, ok := cc.siteOn(t); ok {
		return cc.buildsDoor(def) || cc.buildsWall(def)
	}
	tiles, ok := cc.TileAt(t)
	return ok && cc.isWall(tiles[data.TileLayer_Wall])
}

func (cc *ChunkContainer) isWall(tile data.TileEnum) bool {
	def, ok := cc.atlas[tile]
	return ok && def.layer == data.TileLayer_Wall
}

func (cc *ChunkContainer) buildsWall(def ecs.BlueprintDef) bool {
	return def.Feature() == data.Feature_None && cc.isWall(def.Tile())
}

func (cc *ChunkContainer) buildsDoor(def ecs.BlueprintDef) bool {
	return cc.featureAtlas[def.Feature()].door != nil
}

// Whether the site builds a door that opens for the faction
func (cc *ChunkContainer) siteOpensFor(t utils.Vec2i, faction data.FactionEnum) bool {
	_, s, def, ok := cc.siteOn(t)
	return ok && cc.buildsDoor(def) && faction != data.Faction_None && s.faction == faction
}

// Facing along the walls it's between, which is across unless there's nothing to either side of it but something above
// or below it
func (cc *ChunkContainer) doorFacesVertically(t utils.Vec2i) bool {
	across := cc.joinsDoor(utils.Vec2i{X: t.X - 1, Y: t.Y}) || cc.joinsDoor(utils.Vec2i{X: t.X + 1, Y: t.Y})
	upright := cc.joinsDoor(utils.Vec2i{X: t.X, Y: t.Y - 1}) || cc.joinsDoor(utils.Vec2i{X: t.X, Y: t.Y + 1})
	return !across && upright
}

// Turns the doors next to the tile to face along what's around them now, once something there has changed
func (cc *ChunkContainer) turnDoorsAround(t utils.Vec2i) {
	for _, n := range []utils.Vec2i{{X: t.X - 1, Y: t.Y}, {X: t.X + 1, Y: t.Y}, {X: t.X, Y: t.Y - 1}, {X: t.X, Y: t.Y + 1}} {
		c, def, ok := cc.featureOn(n)
		if !ok || def.door == nil {
			continue
		}
		if vertical := cc.doorFacesVertically(n); vertical != def.door.vertical {
			turned, _ := doorVariant(cc.featureAtlas, def.door.kind, vertical, def.door.open)
			cc.setFeature(c, n, turned)
		}
	}
}

// A door opens as soon as the body of an entity of its faction is on it or next to it, together with the doors it's in a
// row with, facing the same way, and closes once there's none, and nothing in the way of any of them. One without a
// faction is no one's, so it never opens.
func (cc *ChunkContainer) TickDoors(x *ecs.ECSContainer) {
	doors := []utils.Vec2i{}
	for _, c := range cc.chunks {
		for t, f := range c.features {
			if cc.featureAtlas[f].door != nil {
				doors = append(doors, t)
			}
		}
	}
	if len(doors) == 0 {
		return
	}
	// Map order differs from run to run, and which door of a row is looked at first mustn't change the row
	slices.SortFunc(doors, func(a, b utils.Vec2i) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })

	near := map[data.FactionEnum][]utils.Vec2i{}
	for e, f := range ecs.Query1(&x.Factions, ecs.Without(&x.Disabled)) {
		if pos, err := x.WorldPos(e); err == nil && f.Faction() != data.Faction_None {
			near[f.Faction()] = append(near[f.Faction()], x.BodyTiles(e, utils.WorldPosToTilePos(pos))...)
		}
	}

	done := map[utils.Vec2i]bool{}
	for _, start := range doors {
		if done[start] {
			continue
		}
		row := cc.doorRow(start)
		door, _ := cc.DoorAt(start)
		open := false
		for _, t := range row {
			done[t] = true
			open = open || slices.ContainsFunc(near[door.Faction], func(e utils.Vec2i) bool {
				return max(abs(e.X-t.X), abs(e.Y-t.Y)) <= 1
			})
		}
		if !open && door.Open {
			open = slices.ContainsFunc(row, x.IsEntityInTile)
		}
		if open != door.Open {
			for _, t := range row {
				c, def, _ := cc.featureOn(t)
				swapped, _ := doorVariant(cc.featureAtlas, def.door.kind, def.door.vertical, open)
				cc.setFeature(c, t, swapped)
			}
		}
	}
}

// The doors in a row with the one on t, facing the same way and along it, with the same faction
func (cc *ChunkContainer) doorRow(t utils.Vec2i) []utils.Vec2i {
	door, _ := cc.DoorAt(t)
	step := utils.Vec2i{X: 1}
	if door.Vertical {
		step = utils.Vec2i{Y: 1}
	}
	row := []utils.Vec2i{t}
	for _, dir := range []int{-1, 1} {
		for n := (utils.Vec2i{X: t.X + dir*step.X, Y: t.Y + dir*step.Y}); ; n = (utils.Vec2i{X: n.X + dir*step.X, Y: n.Y + dir*step.Y}) {
			other, ok := cc.DoorAt(n)
			if !ok || other.Vertical != door.Vertical || other.Faction != door.Faction {
				break
			}
			row = append(row, n)
		}
	}
	return row
}

func abs(n int) int {
	return max(n, -n)
}
