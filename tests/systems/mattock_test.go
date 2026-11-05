package systems_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"maps"
	"testing"
)

type openGround struct {
	damaged map[utils.Vec2i]int
}

func (openGround) IsOpaqueTile(utils.Vec2i) bool { return false }

func (openGround) IsOpenToSky(utils.Vec2i) bool { return true }

func (openGround) TileLayer(utils.Vec2i) (ecs.LayerMask, bool) { return ecs.Layer_Ground, true }

func (openGround) IsBlockedAt(utils.Vec2f) bool { return false }

func (openGround) CanDamageFeature(utils.Vec2i) bool { return false }

func (openGround) DamageFeature(utils.Vec2i, int) (bool, *ecs.DropTable) { return false, nil }

func (openGround) CanDamageTile(utils.Vec2i) bool { return true }

func (g openGround) DamageTile(t utils.Vec2i, damage int) (bool, *ecs.DropTable) {
	g.damaged[t] += damage
	return true, nil
}

func (openGround) CanPlaceTile(utils.Vec2i) bool { return false }

func (openGround) PlaceTile(utils.Vec2i, data.TileEnum) bool { return false }

func (openGround) CanPlaceFeature(utils.Vec2i) bool { return false }

func (openGround) PlaceFeature(utils.Vec2i, data.FeatureEnum) bool { return false }

func (openGround) LightSources(utils.Vec2i, utils.Vec2i, func(utils.Vec2i, uint8, int)) {}
func (openGround) CanPlaceSite(utils.Vec2i) bool                                        { return false }
func (openGround) PlaceSite(utils.Vec2i, data.BlueprintEnum) bool                       { return false }
func (openGround) Site(utils.Vec2i) (ecs.ConstructionSite, bool) {
	return ecs.ConstructionSite{}, false
}
func (openGround) RemoveSite(utils.Vec2i) bool                           { return false }
func (openGround) DeliverToSite(utils.Vec2i, data.ItemEnum, int) int     { return 0 }
func (openGround) BuildSite(utils.Vec2i, int, bool) (worked, built bool) { return false, false }

func (openGround) WalkSpeedPercent(utils.Vec2i) int { return 100 }

func TestTheMattockStrikesTheTileUnderTheCursor(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)
	ground := openGround{damaged: map[utils.Vec2i]int{}}
	x.SetTerrain(ground)
	spawn := utils.TilePosToWorldPos(utils.Vec2i{X: 10, Y: 10})
	testutil.SetPos(t, x, player, spawn)

	cursor := spawn.Add(utils.Vec2f{X: 30, Y: -5})
	press(t, x, player, ecs.InputState{MainHandEqAbility1: 1, AimPos: cursor, HasAim: true})

	if want := map[utils.Vec2i]int{utils.WorldPosToTilePos(cursor): 50}; !maps.Equal(ground.damaged, want) {
		t.Fatalf("damaged %v, want %v", ground.damaged, want)
	}
}

func TestOnlyATileTargetingItemInHandTargetsTiles(t *testing.T) {
	x, player, _, _, _ := newArmedPlayer(t)

	if _, ok := x.HeldTileTargeting(player); !ok {
		t.Fatal("the mattock in hand doesn't target tiles")
	}
	press(t, x, player, ecs.InputState{SelectSlot: 2})
	if _, ok := x.HeldTileTargeting(player); ok {
		t.Fatal("the bazooka in hand targets tiles")
	}
	press(t, x, player, ecs.InputState{SelectSlot: 1})
	x.SetDisabled(player, true)
	if _, ok := x.HeldTileTargeting(player); ok {
		t.Fatal("a mattock put away with its disabled holder targets tiles")
	}
}
