package ecs_test

import (
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

func TestContactDamageRemembersWhatItHitWhenSaved(t *testing.T) {
	x := testutil.NewECS(t)
	cd := ecs.NewContactDamageComponent(-1, 0, 5)
	e := testutil.AddMovable(t, x, utils.Vec2f{}, false, cd)
	stored, _ := x.ContactDamages.Get(e)
	tile := utils.Vec2i{X: 3, Y: -4}
	stored.RecordEntityHit(7)
	stored.RecordTileHit(tile)
	stored.RecordFeatureHit(tile)

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	restored, ok := testutil.Restore(t, members).ContactDamages.Get(e)
	if !ok {
		t.Fatal("contact damage wasn't restored")
	}

	if restored.CanHitEntity(7) || restored.CanHitTile(tile) || restored.CanHitFeature(tile) {
		t.Fatalf("after restoring, it can hit the entity %v, the tile %v and the feature %v again, want none",
			restored.CanHitEntity(7), restored.CanHitTile(tile), restored.CanHitFeature(tile))
	}
	if !restored.CanHitFeature(utils.Vec2i{}) {
		t.Fatal("after restoring, it can't hit a feature it never hit")
	}
}
