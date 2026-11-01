package ecs_test

import (
	"bytes"
	"ebittest/content/inputsources"
	"ebittest/content/templates"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"encoding/gob"
	"reflect"
	"testing"
)

func TestDtosShareNoMemoryWithLiveComponents(t *testing.T) {
	x := testutil.NewECS(t)
	spr, err := ecs.NewSpriteComponent(common.AssetSheetSlime, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	loop := &inputsources.InputLoopParams{LoopInputs: []ecs.InputState{{Analog1X: 1}, {Analog1X: -1}, {}}}
	p := testutil.Add(t, x, ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.Vec2f{}, 1, 0), spr,
		ecs.NewInputComponent(nil, inputsources.InputType_Loop, loop, ecs.Facing_None))
	c1 := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	c2 := testutil.AddMovable(t, x, utils.Vec2f{}, false)
	testutil.Attach(t, x, c1, p, ecs.Link_Held)
	testutil.Attach(t, x, c2, p, ecs.Link_Held)
	if err := x.SpriteManager.SetSpriteFlash(p, []utils.RelativeColor{{R: 1}, {R: 2}}, []int{100, 100}, 1000, x); err != nil {
		t.Fatal(err)
	}

	dtos := x.GetEntityComponents(p)
	before := encode(t, dtos)

	if err := x.Detach(c1); err != nil {
		t.Fatal(err)
	}
	params, err := x.InputManager.GetParams(p, x)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 10 {
		if err := x.SpriteManager.TickFlash(p, x); err != nil {
			t.Fatal(err)
		}
		if _, err := inputsources.LoopInputSource(p, uint64(i), params, x); err != nil {
			t.Fatal(err)
		}
	}

	if !bytes.Equal(before, encode(t, dtos)) {
		t.Fatal("DTOs changed when the live components did")
	}
}

func TestPlayerSurvivesGobRoundTrip(t *testing.T) {
	x := testutil.NewECS(t)
	p, err := templates.SpawnPlayer(x)
	if err != nil {
		t.Fatal(err)
	}
	if err := templates.SpawnStartingEquipment(x, p); err != nil {
		t.Fatal(err)
	}

	members, err := x.GetCompositionComponents(p)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[common.EntityId][]ecs.ComponentDto
	if err := gob.NewDecoder(bytes.NewReader(encode(t, members))).Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	y := testutil.Restore(t, decoded)
	if err := y.CheckConsistency(); err != nil {
		t.Fatalf("restored player is inconsistent: %v", err)
	}
	for e := range members {
		if got, want := y.GetEntityComponents(e), x.GetEntityComponents(e); !reflect.DeepEqual(got, want) {
			t.Errorf("entity %d changed in the round trip:\n got %#v\nwant %#v", e, got, want)
		}
	}
}

func encode(t *testing.T, v any) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}
