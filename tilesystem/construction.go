package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"slices"
)

type constructionSite struct {
	blueprint data.BlueprintEnum
	// In the order of the blueprint's costs
	delivered []int
	health    int
}

// The tile atlas has to be loaded first, since blueprints build its tiles, and their health is the tile's
func (cc *ChunkContainer) SetBlueprintAtlas(atlas *ecs.BlueprintAtlas) error {
	for _, kind := range atlas.Kinds() {
		def, _ := atlas.Def(kind)
		tile, ok := cc.atlas[def.Tile()]
		if !ok {
			return fmt.Errorf("blueprint %d builds tile %d, which isn't in the tile atlas", kind, def.Tile())
		}
		if tile.maxHealth <= 1 {
			return fmt.Errorf("blueprint %d builds tile %d, whose health %d leaves no room for building it up from 1", kind, def.Tile(), tile.maxHealth)
		}
	}
	cc.blueprints = atlas
	return nil
}

func (cc *ChunkContainer) CanPlaceSite(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}
	if _, hasFeature := c.features[t]; hasFeature {
		return false
	}
	if _, hasSite := c.sites[t]; hasSite {
		return false
	}
	def, ok := cc.atlas[c.getTileDefId(t)]
	return ok && def.buildable
}

func (cc *ChunkContainer) PlaceSite(t utils.Vec2i, blueprint data.BlueprintEnum) bool {
	def, ok := cc.blueprints.Def(blueprint)
	if !ok || !cc.CanPlaceSite(t) {
		return false
	}

	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	c.sites[t] = &constructionSite{blueprint: blueprint, delivered: make([]int, len(def.Costs())), health: 1}
	c.terrainModified = true
	return true
}

func (cc *ChunkContainer) Site(t utils.Vec2i) (ecs.ConstructionSite, bool) {
	_, s, def, ok := cc.siteOn(t)
	if !ok {
		return ecs.ConstructionSite{}, false
	}

	site := ecs.ConstructionSite{Blueprint: s.blueprint, Health: s.health, MaxHealth: cc.atlas[def.Tile()].maxHealth}
	for i, cost := range def.Costs() {
		site.Delivered = append(site.Delivered, ecs.ItemCost{Kind: cost.Kind, Count: s.delivered[i]})
	}
	return site, true
}

func (cc *ChunkContainer) DeliverToSite(t utils.Vec2i, item data.ItemEnum, count int) int {
	c, s, def, ok := cc.siteOn(t)
	if !ok {
		return 0
	}

	taken := 0
	for i, cost := range def.Costs() {
		if cost.Kind != item {
			continue
		}
		n := min(cost.Count-s.delivered[i], count-taken)
		s.delivered[i] += n
		taken += n
	}
	if taken > 0 {
		c.terrainModified = true
	}
	return taken
}

// Each strike adds the share of the tile's health that its work is of the blueprint's, and at least 1, so that the
// work adds up to the tile's health
func (cc *ChunkContainer) BuildSite(t utils.Vec2i, work int) (worked, built bool) {
	c, s, def, ok := cc.siteOn(t)
	if !ok || work <= 0 {
		return false, false
	}
	for i, cost := range def.Costs() {
		if s.delivered[i] < cost.Count {
			return false, false
		}
	}

	maxHealth := cc.atlas[def.Tile()].maxHealth
	s.health += max(1, maxHealth*work/def.Work())
	c.terrainModified = true
	if s.health < maxHealth {
		return true, false
	}

	delete(c.sites, t)
	cc.setTile(c, t, def.Tile())
	return true, true
}

func (cc *ChunkContainer) siteOn(t utils.Vec2i) (*chunk, *constructionSite, ecs.BlueprintDef, bool) {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return nil, nil, ecs.BlueprintDef{}, false
	}
	s, ok := c.sites[t]
	if !ok {
		return nil, nil, ecs.BlueprintDef{}, false
	}
	def, ok := cc.blueprints.Def(s.blueprint)
	return c, s, def, ok
}

// A ghost of the finished tile until building starts, and then the stage it's reached
func (cc *ChunkContainer) siteSprite(t utils.Vec2i, s *constructionSite) (common.StaticSprite, bool) {
	def, ok := cc.blueprints.Def(s.blueprint)
	if !ok {
		return common.StaticSprite{}, false
	}

	sprite := common.StaticSprite{ImageAsset: def.ImageAsset(), Pos: utils.TilePosToWorldPos(t), Layer: data.ConstructionSiteSpriteLayer}
	if s.health <= 1 {
		sprite.Ghost = true
		return sprite, true
	}
	maxHealth := cc.atlas[def.Tile()].maxHealth
	sprite.Frame = 1 + min(def.Stages()-1, s.health*def.Stages()/maxHealth)
	return sprite, true
}

type siteDto struct {
	X         int
	Y         int
	Blueprint data.BlueprintEnum
	Delivered []int
	Health    int
}

// In tile order, so that saving the same chunk twice writes the same file
func siteDtos(c *chunk) []siteDto {
	sites := make([]siteDto, 0, len(c.sites))
	for t, s := range c.sites {
		sites = append(sites, siteDto{X: t.X, Y: t.Y, Blueprint: s.blueprint, Delivered: slices.Clone(s.delivered), Health: s.health})
	}
	slices.SortFunc(sites, func(a, b siteDto) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return sites
}

func sitesFromDtos(dtos []siteDto) map[utils.Vec2i]*constructionSite {
	sites := make(map[utils.Vec2i]*constructionSite, len(dtos))
	for _, d := range dtos {
		sites[utils.Vec2i{X: d.X, Y: d.Y}] = &constructionSite{blueprint: d.Blueprint, delivered: slices.Clone(d.Delivered), health: d.Health}
	}
	return sites
}
