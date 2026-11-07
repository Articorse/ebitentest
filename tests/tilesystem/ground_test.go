package tilesystem_test

import (
	"ebittest/content/abilities"
	"ebittest/data"
	"ebittest/ecs"
	ecscommon "ebittest/ecs/common"
	"ebittest/ecs/systems/drawsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"slices"
	"testing"
)

// How many of the kind lie on the ground as far around the tile as things spill
func lyingAround(s *testutil.Session, kind data.ItemEnum, near utils.Vec2i) int {
	r := data.GroundSpillRangeTiles
	n := 0
	for _, t := range s.Chunks.GroundStacksIn(utils.Vec2i{X: near.X - r, Y: near.Y - r}, utils.Vec2i{X: near.X + r, Y: near.Y + r}) {
		if stack, _ := s.Chunks.GroundStack(t); stack.Unique == nil && stack.Kind == kind {
			n += stack.Count
		}
	}
	return n
}

func onGround(s *testutil.Session, t utils.Vec2i) (data.ItemEnum, int) {
	stack, ok := s.Chunks.GroundStack(t)
	if !ok {
		return data.Item_None, 0
	}
	return stack.Kind, stack.Count
}

// A row of open ground, first tile left
func openRow(t *testing.T, s *testutil.Session, colonist ecscommon.EntityId, n int) utils.Vec2i {
	t.Helper()
	return storageSpot(t, s, colonist, 6, n, 1)
}

func TestWhatsDroppedJoinsTheStackThereThenSpillsOntoTheClosestEmptyGround(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 3)
	right := utils.Vec2i{X: row.X + 1, Y: row.Y}

	if laid := s.ECS.LayDownKind(data.Item_Wood, 90, right); laid != 90 {
		t.Fatalf("laid down %d of 90 wood", laid)
	}
	if laid := s.ECS.LayDownKind(data.Item_Wood, 30, right); laid != 30 {
		t.Fatalf("laid down %d of 30 more wood", laid)
	}
	if kind, n := onGround(s, right); kind != data.Item_Wood || n != 100 {
		t.Fatalf("the tile it was dropped on holds %d of %v, want a full stack of 100 wood", n, kind)
	}
	spilled := 0
	for _, tile := range s.Chunks.GroundStacksIn(utils.Vec2i{X: right.X - 1, Y: right.Y - 1}, utils.Vec2i{X: right.X + 1, Y: right.Y + 1}) {
		if tile != right {
			_, n := onGround(s, tile)
			spilled += n
		}
	}
	if spilled != 20 {
		t.Fatalf("%d wood spilled next to the full stack, want 20", spilled)
	}

	if laid := s.ECS.LayDownKind(data.Item_Torch, 2, right); laid != 2 {
		t.Fatal("torches dropped onto wood didn't go next to it")
	}
	if kind, n := onGround(s, right); kind != data.Item_Wood || n != 100 {
		t.Fatal("torches went onto a tile of wood")
	}
}

func TestWhatsDroppedStaysAsCloseAsItCanRatherThanJoiningStacksFurtherAway(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 3)
	far := utils.Vec2i{X: row.X + 2, Y: row.Y}
	s.ECS.LayDownKind(data.Item_Wood, 50, far)

	s.ECS.LayDownKind(data.Item_Wood, 10, row)
	if _, n := onGround(s, row); n != 10 {
		t.Fatalf("the tile wood was dropped on holds %d, want the 10 dropped", n)
	}
	if _, n := onGround(s, far); n != 50 {
		t.Fatalf("the stack two tiles away holds %d, want its 50 left alone", n)
	}
}

func TestNothingLiesOnGroundThatCantBeWalkedOn(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	tree := treeAway(t, s, colonist, 6)
	s.ECS.LayDownKind(data.Item_Wood, 5, tree)
	if _, ok := s.Chunks.GroundStack(tree); ok {
		t.Fatal("wood was laid down on a tree")
	}
	found := false
	for _, tile := range s.Chunks.GroundStacksIn(utils.Vec2i{X: tree.X - 1, Y: tree.Y - 1}, utils.Vec2i{X: tree.X + 1, Y: tree.Y + 1}) {
		_, n := onGround(s, tile)
		found = found || n == 5
	}
	if !found {
		t.Fatal("the wood didn't spill next to the tree")
	}
}

func TestAGenericItemLaidDownLeavesItsHolder(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	inv, _ := s.ECS.Inventories.Get(colonist)
	wood, _ := inv.Slot(0)
	before := countOf(s, wood)

	if ok, err := s.ECS.LayDown(wood, row); err != nil || !ok {
		t.Fatalf("the colonist's wood wasn't laid down: %v", err)
	}
	if s.ECS.HasEntity(wood) || woodCarried(s, colonist) != 0 {
		t.Fatal("the wood laid down is still an entity, or still carried")
	}
	if kind, n := onGround(s, row); kind != data.Item_Wood || n != before {
		t.Fatalf("the tile holds %d of %v, want the %d wood", n, kind, before)
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestAUniqueItemIsKeptWholeOnItsOwnTileAndComesBackTheSame(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 2)
	s.ECS.LayDownKind(data.Item_Wood, 3, row)
	axe, _ := colonistAxe(s, colonist)

	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("the axe wasn't laid down: %v", err)
	}
	if s.ECS.HasEntity(axe) {
		t.Fatal("the axe laid down is still an entity")
	}
	next, stack, ok := utils.Vec2i{}, ecs.GroundStack{}, false
	for _, tile := range s.Chunks.GroundStacksIn(utils.Vec2i{X: row.X - 1, Y: row.Y - 1}, utils.Vec2i{X: row.X + 1, Y: row.Y + 1}) {
		if st, _ := s.Chunks.GroundStack(tile); st.Unique != nil {
			next, stack, ok = tile, st, true
		}
	}
	if !ok || stack.Unique == nil || stack.Unique.Name != "Axe" || stack.Unique.Image != ecscommon.AssetImageAxe {
		t.Fatalf("the axe isn't on a tile next to the wood as itself: %+v", stack)
	}
	if lines := drawsystem.GroundStackTooltip(stack, s.ECS); !slices.Equal(lines, []string{"Axe"}) {
		t.Fatalf("the axe's tooltip says %q", lines)
	}

	if n, err := s.ECS.PickUpFromGround(colonist, next); err != nil || n != 1 {
		t.Fatalf("picked up %d of the axe: %v", n, err)
	}
	if _, ok := s.Chunks.GroundStack(next); ok {
		t.Fatal("the axe is still on the ground")
	}
	if back, ok := colonistAxe(s, colonist); !ok || back != axe {
		t.Fatal("the axe didn't come back as the same entity, in the colonist's inventory")
	}
	if eq, _ := s.ECS.Equipments.Get(axe); !eq.HasAbility(abilities.Ability_Slash) {
		t.Fatal("the axe came back without its slash")
	}
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestPickingUpTakesWhatThereIsRoomForAndLeavesTheRest(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	holder, err := s.ECS.AddNewEntity(ecs.NewHierarchyComponent(), ecs.NewTransformComponent(utils.TilePosToWorldPos(row), 1, 0), ecs.NewInventoryComponent(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	wood, _ := s.ECS.SpawnItemStack(data.Item_Wood, 70, utils.Vec2f{})
	if err := s.ECS.PutInInventory(holder, 0, wood); err != nil {
		t.Fatal(err)
	}
	s.ECS.LayDownKind(data.Item_Wood, 50, row)

	if n, err := s.ECS.PickUpFromGround(holder, row); err != nil || n != 30 {
		t.Fatalf("picked up %d wood, want the 30 there's room for: %v", n, err)
	}
	if _, n := onGround(s, row); n != 20 || countOf(s, wood) != 100 {
		t.Fatalf("%d wood was left and %d held, want 20 and 100", n, countOf(s, wood))
	}

	s.ECS.LayDownKind(data.Item_Torch, 1, row)
	if n, _ := s.ECS.PickUpFromGround(holder, row); n != 0 {
		t.Fatal("a full inventory picked something up")
	}

	if n, _ := s.ECS.PickUpFromGround(colonist, row); n != 20 {
		t.Fatalf("the colonist picked up %d of the 20 wood", n)
	}
	if _, ok := s.Chunks.GroundStack(row); ok {
		t.Fatal("picking up all of a stack left it on the ground")
	}
	testutil.RemoveScheduled(t, s.ECS)
	if err := s.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestWhatLiesOnTheGroundIsSavedWithItsChunk(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 1)
	s.ECS.LayDownKind(data.Item_Wood, 42, row)
	axe, _ := colonistAxe(s, colonist)
	if ok, err := s.ECS.LayDown(axe, row); err != nil || !ok {
		t.Fatalf("couldn't lay down the axe: %v", err)
	}
	s.Commit()

	r := testutil.StartSession(t)
	if kind, n := onGround(r, row); kind != data.Item_Wood || n != 42 {
		t.Fatalf("after restarting, the tile holds %d of %v, want 42 wood", n, kind)
	}
	found := false
	for _, tile := range r.Chunks.GroundStacksIn(utils.Vec2i{X: row.X - 1, Y: row.Y - 1}, utils.Vec2i{X: row.X + 1, Y: row.Y + 1}) {
		if stack, _ := r.Chunks.GroundStack(tile); stack.Unique != nil && stack.Unique.Root == axe {
			found = true
			if n, err := r.ECS.PickUpFromGround(colonist, tile); err != nil || n != 1 {
				t.Fatalf("the axe couldn't be picked back up after restarting: %v", err)
			}
		}
	}
	if !found {
		t.Fatal("the axe wasn't saved")
	}
	if back, ok := colonistAxe(r, colonist); !ok || back != axe {
		t.Fatal("the axe picked up after restarting isn't the colonist's axe")
	}
	if err := r.ECS.CheckConsistency(); err != nil {
		t.Fatal(err)
	}
}

func TestWhatLiesOnATileIsDrawnThereInTileOrderAndBlocksBuilding(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	row := openRow(t, s, colonist, 3)
	far := utils.Vec2i{X: row.X + 2, Y: row.Y}
	s.ECS.LayDownKind(data.Item_Torch, 1, far)
	s.ECS.LayDownKind(data.Item_Wood, 7, row)

	sprites := drawsystem.GroundItemSprites([2]utils.Vec2f{utils.TilePosToWorldPos(row), utils.TilePosToWorldPos(far)}, s.ECS)
	if len(sprites) != 2 || sprites[0].Pos != utils.TilePosToWorldPos(row) || sprites[1].Pos != utils.TilePosToWorldPos(far) {
		t.Fatalf("what's on the ground is drawn as %+v", sprites)
	}
	woodDef, _ := testutil.ItemAtlas(t).Def(data.Item_Wood)
	if sprites[0].ImageAsset != woodDef.ImageAsset() {
		t.Fatal("the wood isn't drawn as wood")
	}
	stack, _ := s.Chunks.GroundStack(row)
	if lines := drawsystem.GroundStackTooltip(stack, s.ECS); !slices.Equal(lines, []string{"7 Wood"}) {
		t.Fatalf("the wood's tooltip says %q", lines)
	}
	if !s.ECS.IsTileOccupied(row) || s.ECS.IsTileOccupied(utils.Vec2i{X: row.X + 1, Y: row.Y}) {
		t.Fatal("a tile with something on it doesn't count as occupied, or an empty one does")
	}
}

func groundTorches(n int) ecs.GroundStack { return ecs.GroundStack{Kind: data.Item_Torch, Count: n} }
func groundWood(n int) ecs.GroundStack    { return ecs.GroundStack{Kind: data.Item_Wood, Count: n} }
