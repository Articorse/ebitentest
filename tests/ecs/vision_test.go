package ecs_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"testing"
)

func TestVisionRadiusChangesAtTheNextUpdate(t *testing.T) {
	v := ecs.NewVisionComponent(3)
	open := func(utils.Vec2i) bool { return false }
	far := utils.Vec2i{X: 5, Y: 1}

	v.Update(utils.Vec2i{}, open)
	if v.CanSee(far) {
		t.Fatal("sees beyond its radius")
	}

	v.SetRadius(6)
	if v.CanSee(far) {
		t.Fatal("the new radius applied before an update")
	}
	v.Update(utils.Vec2i{}, open)
	if !v.CanSee(far) {
		t.Fatal("the new radius didn't apply at the update")
	}

	v.SetRadius(1)
	v.Update(utils.Vec2i{X: 100}, open)
	if v.CanSee(far) || !v.CanSee(utils.Vec2i{X: 101}) {
		t.Fatal("what's visible didn't follow the new origin and radius")
	}
}

func TestVisionSavesOnlyItsRadius(t *testing.T) {
	x := testutil.NewECS(t)
	e := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), ecs.NewVisionComponent(4))
	v, _ := x.Visions.Get(e)
	v.Update(utils.Vec2i{}, func(utils.Vec2i) bool { return false })

	members, err := x.GetCompositionComponents(e)
	if err != nil {
		t.Fatal(err)
	}
	y := testutil.Restore(t, members)

	restored, ok := y.Visions.Get(e)
	if !ok || restored.Radius() != 4 {
		t.Fatalf("restored vision %v with radius %d, want radius 4", ok, restored.Radius())
	}
	if restored.CanSee(utils.Vec2i{}) {
		t.Fatal("restored vision sees something before its first update")
	}
}

func TestVisionCircleOnlyShowsComputedTiles(t *testing.T) {
	const radius = data.PlayerVisionRadiusTiles
	outer := (radius - data.VisionCircleMarginTiles) * data.TileSize
	v := ecs.NewVisionComponent(radius)

	for i := range 9 {
		for j := range 9 {
			// Positions spread over the whole tile, edges included
			viewer := utils.Vec2f{X: (float64(i)/8 - 0.5) * data.TileSize, Y: (float64(j)/8 - 0.5) * data.TileSize}
			v.Update(utils.WorldPosToTilePos(viewer), func(utils.Vec2i) bool { return false })

			for a := range 360 {
				angle := float64(a) * math.Pi / 180
				edge := viewer.Add(utils.Vec2f{X: math.Cos(angle) * outer, Y: math.Sin(angle) * outer})
				if tile := utils.WorldPosToTilePos(edge); !v.CanSee(tile) {
					t.Fatalf("viewer at %v: the circle reaches tile %v, which wasn't computed as visible", viewer, tile)
				}
			}
		}
	}
}
