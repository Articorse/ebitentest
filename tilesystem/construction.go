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

// A blueprint until the first material is delivered, then a marker while it collects the rest, and under construction
// once it has them all
type constructionSite struct {
	blueprint data.BlueprintEnum
	// In the order of the blueprint's costs
	delivered []int
	health    int
	order     ecs.SiteOrder
	// Whose what's built is, for what tells
	faction data.FactionEnum
}

// Only a plan, which isn't hit by anything, until something's been delivered to it
func (s *constructionSite) isBlueprint() bool {
	for _, n := range s.delivered {
		if n > 0 {
			return false
		}
	}
	return true
}

func (s *constructionSite) isSupplied(def ecs.BlueprintDef) bool {
	for i, cost := range def.Costs() {
		if s.delivered[i] < cost.Amount {
			return false
		}
	}
	return true
}

// The tile and feature atlases have to be loaded first, since blueprints build their tiles and features, and their
// health is the tile's or feature's
func (cc *ChunkContainer) SetBlueprintAtlas(atlas *ecs.BlueprintAtlas) error {
	for _, kind := range atlas.Kinds() {
		def, _ := atlas.Def(kind)
		if def.Feature() != data.Feature_None {
			feature, ok := cc.featureAtlas[def.Feature()]
			if !ok {
				return fmt.Errorf("blueprint %d builds feature %d, which isn't in the feature atlas", kind, def.Feature())
			}
			if feature.maxHealth <= 1 {
				return fmt.Errorf("blueprint %d builds feature %d, whose health %d leaves no room for building it up from 1", kind, def.Feature(), feature.maxHealth)
			}
			continue
		}
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

func (cc *ChunkContainer) siteMaxHealth(def ecs.BlueprintDef) int {
	if def.Feature() != data.Feature_None {
		return cc.featureAtlas[def.Feature()].maxHealth
	}
	return cc.atlas[def.Tile()].maxHealth
}

func (cc *ChunkContainer) CanPlaceSite(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return false
	}
	if _, hasFeature := c.features[t]; hasFeature {
		return false
	}
	if _, hasSite := c.sites[t]; hasSite || cc.InArea(data.Area_Storage, t) {
		return false
	}
	return c.propsAt(t).buildable
}

// Where a site can be placed, or a door in place of a wall blueprint, which is a wall with a door put in it, as long as
// nothing's been delivered to the wall that would be lost
func (cc *ChunkContainer) CanPlaceBlueprint(t utils.Vec2i, blueprint data.BlueprintEnum) bool {
	def, ok := cc.blueprints.Def(blueprint)
	if !ok {
		return false
	}
	if cc.CanPlaceSite(t) {
		return true
	}
	_, s, under, ok := cc.siteOn(t)
	return ok && cc.buildsDoor(def) && cc.buildsWall(under) && s.isBlueprint()
}

func (cc *ChunkContainer) PlaceSite(t utils.Vec2i, blueprint data.BlueprintEnum) bool {
	def, ok := cc.blueprints.Def(blueprint)
	if !ok || !cc.CanPlaceBlueprint(t, blueprint) {
		return false
	}

	c := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	c.sites[t] = &constructionSite{blueprint: blueprint, delivered: make([]int, len(def.Costs())), health: 1}
	c.terrainModified = true
	cc.turnDoorsAround(t)
	return true
}

func (cc *ChunkContainer) SetSiteFaction(t utils.Vec2i, faction data.FactionEnum) bool {
	_, s, _, ok := cc.siteOn(t)
	if ok {
		s.faction = faction
	}
	return ok
}

func (cc *ChunkContainer) Site(t utils.Vec2i) (ecs.ConstructionSite, bool) {
	_, s, def, ok := cc.siteOn(t)
	if !ok {
		return ecs.ConstructionSite{}, false
	}

	site := ecs.ConstructionSite{Blueprint: s.blueprint, Health: s.health, MaxHealth: cc.siteMaxHealth(def), Order: s.order, Faction: s.faction}
	for i, cost := range def.Costs() {
		site.Delivered = append(site.Delivered, ecs.ItemCost{Kind: cost.Kind, Amount: s.delivered[i]})
	}
	return site, true
}

func (cc *ChunkContainer) SetSiteOrder(t utils.Vec2i, order ecs.SiteOrder) bool {
	_, s, _, ok := cc.siteOn(t)
	if ok {
		s.order = order
	}
	return ok
}

func (cc *ChunkContainer) DeliverToSite(t utils.Vec2i, item data.ItemEnum, amount int) int {
	c, s, def, ok := cc.siteOn(t)
	if !ok {
		return 0
	}

	taken := 0
	for i, cost := range def.Costs() {
		if cost.Kind != item {
			continue
		}
		n := min(cost.Amount-s.delivered[i], amount-taken)
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
func (cc *ChunkContainer) BuildSite(t utils.Vec2i, work int, canFinish bool) (worked, built bool) {
	c, s, def, ok := cc.siteOn(t)
	if !ok || work <= 0 || !s.isSupplied(def) {
		return false, false
	}

	maxHealth := cc.siteMaxHealth(def)
	health := s.health + ecs.SiteWorkStep(def, maxHealth, work)
	if health >= maxHealth && !canFinish {
		return false, false
	}
	s.health = health
	c.terrainModified = true
	if s.health < maxHealth {
		return true, false
	}

	delete(c.sites, t)
	if def.Feature() == data.Feature_None {
		cc.setTile(c, t, cc.atlas[def.Tile()].layer, def.Tile())
		return true, true
	}
	feature := def.Feature()
	if door := cc.featureAtlas[feature].door; door != nil {
		feature, _ = doorVariant(cc.featureAtlas, door.kind, cc.doorFacesVertically(t), false)
		c.doorFactions[t] = s.faction
	}
	cc.setFeature(c, t, feature)
	cc.turnDoorsAround(t)
	return true, true
}

func (cc *ChunkContainer) SitesIn(first, last utils.Vec2i) []utils.Vec2i {
	tiles := []utils.Vec2i{}
	for cy := floorDiv(first.Y, data.ChunkSize); cy <= floorDiv(last.Y, data.ChunkSize); cy++ {
		for cx := floorDiv(first.X, data.ChunkSize); cx <= floorDiv(last.X, data.ChunkSize); cx++ {
			c, ok := cc.chunks[utils.Vec2i{X: cx, Y: cy}]
			if !ok {
				continue
			}
			for t := range c.sites {
				if t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y {
					tiles = append(tiles, t)
				}
			}
		}
	}
	// Map order differs from run to run, and what colonists choose mustn't
	slices.SortFunc(tiles, func(a, b utils.Vec2i) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return tiles
}

// Only a blueprint can be taken away like this, since a site with materials has to be broken to get them back
func (cc *ChunkContainer) RemoveSite(t utils.Vec2i) bool {
	c, s, _, ok := cc.siteOn(t)
	if !ok || !s.isBlueprint() {
		return false
	}
	delete(c.sites, t)
	c.terrainModified = true
	cc.turnDoorsAround(t)
	return true
}

// A site with materials is hit instead of the ground under it, and drops all of them when it's destroyed. Ok is false
// where there's no such site, and the ground is hit instead.
func (cc *ChunkContainer) damageSite(t utils.Vec2i, damage int) (drops *ecs.DropTable, ok bool) {
	c, s, def, ok := cc.siteOn(t)
	if !ok || s.isBlueprint() {
		return nil, false
	}

	s.health -= damage
	c.terrainModified = true
	if s.health > 0 {
		return nil, true
	}

	delete(c.sites, t)
	cc.turnDoorsAround(t)
	items := []ecs.ItemDrop{}
	for i, cost := range def.Costs() {
		if s.delivered[i] > 0 {
			items = append(items, ecs.ItemDrop{Kind: cost.Kind, Amount: s.delivered[i]})
		}
	}
	table, err := ecs.NewDropTable(ecs.DropOutcome{ChancePercent: 100, Items: items})
	if err != nil {
		return nil, true
	}
	return table, true
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

// The finished tile as a blueprint until something's delivered, a marker while it's collecting the rest, and then the
// stage building has reached, starting at the first as soon as it has everything
// The sprites of the blueprints within the bounds, in tile order. They're drawn apart from features and other sites,
// above the light and fog, since they're only a plan.
func (cc *ChunkContainer) BlueprintSprites(bounds [2]utils.Vec2f) []common.StaticSprite {
	minTile, maxTile := utils.WorldPosToTilePos(bounds[0]), utils.WorldPosToTilePos(bounds[1])
	sprites := []common.StaticSprite{}
	for _, t := range cc.SitesIn(minTile, maxTile) {
		_, site, _, _ := cc.siteOn(t)
		if !site.isBlueprint() {
			continue
		}
		if sprite, ok := cc.siteSprite(t, site); ok {
			sprites = append(sprites, sprite)
		}
	}
	return sprites
}

func (cc *ChunkContainer) siteSprite(t utils.Vec2i, s *constructionSite) (common.StaticSprite, bool) {
	def, ok := cc.blueprints.Def(s.blueprint)
	if !ok {
		return common.StaticSprite{}, false
	}

	sprite := common.StaticSprite{ImageAsset: def.ImageAsset(), Pos: utils.TilePosToWorldPos(t), Layer: data.ConstructionSiteSpriteLayer}
	switch {
	case s.isBlueprint():
		sprite.ImageAsset, sprite.Frame = def.Ghost()
		if door := cc.featureAtlas[def.Feature()].door; door != nil {
			closed, _ := doorVariant(cc.featureAtlas, door.kind, cc.doorFacesVertically(t), false)
			feature, _ := cc.featureSprite(t, closed, 0)
			sprite.ImageAsset, sprite.Frame, sprite.Pos = feature.ImageAsset, feature.Frame, feature.Pos
		}
		sprite.Ghost = true
		return sprite, true
	case !s.isSupplied(def):
		sprite.ImageAsset = common.AssetImageTileMarker
		return sprite, true
	}
	maxHealth := cc.siteMaxHealth(def)
	sprite.Frame = 1 + min(def.Stages()-1, s.health*def.Stages()/maxHealth)
	return sprite, true
}

type siteDto struct {
	X         int
	Y         int
	Blueprint data.BlueprintEnum
	Delivered []int
	Health    int
	Order     ecs.SiteOrder
	Faction   data.FactionEnum
}

// In tile order, so that saving the same chunk twice writes the same file
func siteDtos(c *chunk) []siteDto {
	sites := make([]siteDto, 0, len(c.sites))
	for t, s := range c.sites {
		sites = append(sites, siteDto{X: t.X, Y: t.Y, Blueprint: s.blueprint, Delivered: slices.Clone(s.delivered), Health: s.health, Order: s.order, Faction: s.faction})
	}
	slices.SortFunc(sites, func(a, b siteDto) int {
		return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
	})
	return sites
}

func sitesFromDtos(dtos []siteDto) map[utils.Vec2i]*constructionSite {
	sites := make(map[utils.Vec2i]*constructionSite, len(dtos))
	for _, d := range dtos {
		sites[utils.Vec2i{X: d.X, Y: d.Y}] = &constructionSite{blueprint: d.Blueprint, delivered: slices.Clone(d.Delivered), health: d.Health, order: d.Order, faction: d.Faction}
	}
	return sites
}
