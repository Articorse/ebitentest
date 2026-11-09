package tilesystem_test

import (
	"ebittest/data"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"slices"
	"testing"
)

// The middle of a ring of tiles r away that walls can go on, and that can be walked round, nearest the start that's
// as the condition wants
func ringSpot(t *testing.T, s *testutil.Session, from utils.Vec2i, r int, also func(middle utils.Vec2i) bool) utils.Vec2i {
	t.Helper()
	for d := 0; d <= 120; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(abs(dx), abs(dy)) != d {
					continue
				}
				middle := at(from, dx, dy)
				if also(middle) && !slices.ContainsFunc(ring(middle, r), func(tile utils.Vec2i) bool { return !s.Chunks.CanPlaceTile(tile) }) {
					return middle
				}
			}
		}
	}
	t.Fatalf("no ring %d tiles round that walls can go on near %v", r, from)
	return utils.Vec2i{}
}

func anywhere(utils.Vec2i) bool { return true }

func roomAt(t *testing.T, s *testutil.Session, tile utils.Vec2i) uint64 {
	t.Helper()
	id, ok := s.Chunks.RoomAt(tile)
	if !ok {
		t.Fatalf("%v isn't in a room", tile)
	}
	return id
}

func TestAWalledInSpaceIsARoomWhetherOrNotADoorLeadsIntoIt(t *testing.T) {
	for _, c := range []struct {
		name string
		gap  func(t *testing.T, s *testutil.Session, gap utils.Vec2i)
		room bool
	}{
		{"walled in", func(t *testing.T, s *testutil.Session, gap utils.Vec2i) { wall(t, s, gap) }, true},
		{"with a door", func(t *testing.T, s *testutil.Session, gap utils.Vec2i) { buildDoor(t, s, gap, data.Faction_Colony) }, true},
		{"with a gap", func(*testing.T, *testutil.Session, utils.Vec2i) {}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			testutil.TempWorld(t)
			s := testutil.StartSession(t)
			middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, func(middle utils.Vec2i) bool {
				return s.Chunks.CanPlaceTile(at(middle, 0, 4))
			})
			gap := at(middle, 0, 3)
			wall(t, s, ring(middle, 3, gap)...)
			c.gap(t, s, gap)

			id, ok := s.Chunks.RoomAt(middle)
			if ok != c.room {
				t.Fatalf("the middle is in a room %v, of %d tiles, want %v", ok, s.Chunks.RoomSize(id), c.room)
			}
			if ok && s.Chunks.RoomSize(id) != 25 {
				t.Fatalf("the room is %d tiles, want the 25 inside the walls", s.Chunks.RoomSize(id))
			}
			if _, ok := s.Chunks.RoomAt(at(middle, 5, 0)); ok {
				t.Fatal("the ground outside is in a room")
			}
			if _, ok := s.Chunks.RoomAt(at(middle, 3, 0)); ok {
				t.Fatal("a wall is in a room")
			}
			if _, ok := s.Chunks.RoomAt(gap); ok {
				t.Fatal("the gap is in a room")
			}
		})
	}
}

func TestADoorBetweenTwoRoomsKeepsThemApart(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, anywhere)
	wall(t, s, ring(middle, 3)...)
	wall(t, s, at(middle, -2, 0), at(middle, -1, 0), at(middle, 1, 0), at(middle, 2, 0))
	buildDoor(t, s, middle, data.Faction_Colony)

	above, below := roomAt(t, s, at(middle, 0, -1)), roomAt(t, s, at(middle, 0, 1))
	if above == below {
		t.Fatal("the two sides of the door are one room")
	}
	if s.Chunks.RoomSize(above) != 10 || s.Chunks.RoomSize(below) != 10 {
		t.Fatalf("the rooms are %d and %d tiles, want 10 each", s.Chunks.RoomSize(above), s.Chunks.RoomSize(below))
	}
}

func TestOnlyUpToAThousandTilesWalledInIsARoom(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	both := func(middle utils.Vec2i) bool {
		return !slices.ContainsFunc(ring(middle, 17), func(tile utils.Vec2i) bool { return !s.Chunks.CanPlaceTile(tile) })
	}
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 16, both)
	wall(t, s, ring(middle, 17)...)
	if _, ok := s.Chunks.RoomAt(middle); ok {
		t.Fatal("33 by 33 tiles walled in is a room")
	}
	wall(t, s, ring(middle, 16)...)
	if id := roomAt(t, s, middle); s.Chunks.RoomSize(id) != 31*31 {
		t.Fatalf("31 by 31 tiles walled in is a room of %d tiles", s.Chunks.RoomSize(id))
	}
}

func TestARoomKeepsItsIdWhereMostOfItCarriesOn(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, anywhere)
	wall(t, s, ring(middle, 3)...)
	id := roomAt(t, s, middle)

	wall(t, s, at(middle, -2, 1), at(middle, -1, 1), at(middle, 0, 1), at(middle, 1, 1), at(middle, 2, 1))
	if roomAt(t, s, middle) != id {
		t.Fatal("the bigger part of the split room didn't keep its id")
	}
	small := roomAt(t, s, at(middle, 0, 2))
	if small == id {
		t.Fatal("the smaller part of the split room has the same id")
	}

	if destroyed, _ := s.Chunks.DamageTile(at(middle, 0, 1), 1000); !destroyed {
		t.Fatal("the wall between them couldn't be knocked down")
	}
	if roomAt(t, s, at(middle, 0, 2)) != id {
		t.Fatal("the joined rooms didn't keep the bigger one's id")
	}
	if s.Chunks.RoomSize(id) != 21 {
		t.Fatalf("the joined room is %d tiles, want the 21 round the walls left", s.Chunks.RoomSize(id))
	}
}

func TestRoomsAreSavedAndShownInTheTooltip(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	middle := ringSpot(t, s, utils.WorldPosToTilePos(testutil.PlayerStart), 3, anywhere)
	wall(t, s, ring(middle, 3)...)
	id := roomAt(t, s, middle)
	want := fmt.Sprintf("Room %d: 25 tiles, 25 unroofed", id)
	if lines := s.Chunks.RoomDebugLines(middle); !slices.Contains(lines, want) {
		t.Fatalf("the tooltip shows %v, want %q", lines, want)
	}
	if lines := s.Chunks.RoomDebugLines(at(middle, 5, 0)); !slices.Contains(lines, "Outdoors") {
		t.Fatalf("the tooltip outside shows %v, want outdoors", lines)
	}
	// Split, so that the smaller part is the first of them, which would be numbered first if they were worked out anew
	wall(t, s, at(middle, -2, -1), at(middle, -1, -1), at(middle, 0, -1), at(middle, 1, -1), at(middle, 2, -1))
	small := roomAt(t, s, at(middle, 0, -2))
	s.Commit()

	r := testutil.StartSession(t)
	if roomAt(t, r, middle) != id || roomAt(t, r, at(middle, 0, -2)) != small {
		t.Fatal("the rooms have different ids after a restart")
	}
	wall(t, r, ring(middle, 1, at(middle, -1, -1), at(middle, 0, -1), at(middle, 1, -1))...)
	if made := roomAt(t, r, middle); made == id || made == small {
		t.Fatal("a room made after a restart was given the id of one saved")
	}
}

func TestARoomPartlyInAChunkThatsLeftStaysWhole(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	edge := data.ChunkSize
	start := utils.Vec2i{X: edge, Y: utils.WorldPosToTilePos(testutil.PlayerStart).Y + 20}
	middle := ringSpot(t, s, start, 3, func(middle utils.Vec2i) bool { return middle.X == edge })
	// Not asked about until the left half has gone, so that it's only worked out as it goes
	wall(t, s, ring(middle, 3)...)

	// The room's left half is in the chunk left, and its right half in one still active
	s.MovePlayer(utils.TilePosToWorldPos(utils.Vec2i{X: (testutil.PlayerLoaderRadius + 2) * data.ChunkSize, Y: middle.Y}))
	s.Tick(5)
	if _, active := s.Chunks.TileAt(at(middle, -1, 0)); active {
		t.Fatal("the room's left half is still active")
	}
	id := roomAt(t, s, at(middle, 1, 0))
	if s.Chunks.RoomSize(id) != 25 {
		t.Fatalf("with its left half gone, the room is %d tiles, want 25", s.Chunks.RoomSize(id))
	}
	wall(t, s, at(middle, 1, -2), at(middle, 1, -1), at(middle, 1, 0), at(middle, 1, 1), at(middle, 1, 2))
	if roomAt(t, s, at(middle, 2, 0)) == id || s.Chunks.RoomSize(id) != 15 {
		t.Fatalf("walling off the active part left the room %d tiles, want the 15 that's left", s.Chunks.RoomSize(id))
	}
	s.Commit()

	r := testutil.StartSession(t)
	r.MovePlayer(utils.TilePosToWorldPos(middle))
	r.Tick(5)
	if roomAt(t, r, at(middle, -1, 0)) != id || r.Chunks.RoomSize(id) != 15 {
		t.Fatalf("back in the left half after a restart, it's room %d of %d tiles, want %d of 15", roomAt(t, r, at(middle, -1, 0)), r.Chunks.RoomSize(id), id)
	}
}

func TestARoomOnHomeIsRoofedOverOnceItsWalledIn(t *testing.T) {
	s, _, colonist := colonistAtWork(t)
	middle := doorYard(t, s, colonist)
	gap := at(middle, 0, 3)
	wall(t, s, ring(middle, 3, gap)...)
	s.ECS.MarkArea(data.Area_NoRoof, at(middle, 1, 1), at(middle, 1, 1), true)
	s.Tick(1)
	if s.Chunks.InArea(data.Area_Roof, middle) {
		t.Fatal("ground that isn't walled in was put on the roof area")
	}
	buildDoor(t, s, gap, data.Faction_Colony)
	s.Tick(1)
	for _, tile := range rect(at(middle, -3, -3), at(middle, 3, 3)) {
		want := tile != at(middle, 1, 1)
		if got := s.Chunks.InArea(data.Area_Roof, tile); got != want {
			t.Fatalf("%v is on the roof area %v, want %v", tile, got, want)
		}
	}
	if s.Chunks.InArea(data.Area_Roof, at(middle, 4, 0)) {
		t.Fatal("the ground outside the walls was put on the roof area")
	}

	s.ECS.MarkArea(data.Area_Roof, middle, middle, false)
	wall(t, s, at(middle, -2, 1), at(middle, -1, 1), at(middle, 0, 1), at(middle, 1, 1), at(middle, 2, 1))
	s.Tick(1)
	if s.Chunks.InArea(data.Area_Roof, middle) {
		t.Fatal("the roof area taken off a tile of the room came back when the room changed")
	}
}

func TestARoomOffHomeIsntRoofedOver(t *testing.T) {
	s, _, colonist := colonistAtWorkWithoutHome(t)
	middle := doorYard(t, s, colonist)
	wall(t, s, ring(middle, 3)...)
	s.Tick(1)
	if s.Chunks.InArea(data.Area_Roof, middle) {
		t.Fatal("a room off home was put on the roof area")
	}
}

func TestWhatsWalledInWithAChunkThatsNeverBeenPartOfARoomIsntARoom(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	edge := data.ChunkSize
	start := utils.Vec2i{X: edge, Y: utils.WorldPosToTilePos(testutil.PlayerStart).Y + 20}
	middle := ringSpot(t, s, start, 3, func(middle utils.Vec2i) bool { return middle.X == edge })
	gap := at(middle, 3, 0)
	wall(t, s, ring(middle, 3, gap)...)

	s.MovePlayer(utils.TilePosToWorldPos(utils.Vec2i{X: (testutil.PlayerLoaderRadius + 2) * data.ChunkSize, Y: middle.Y}))
	s.Tick(5)
	wall(t, s, gap)
	if _, ok := s.Chunks.RoomAt(at(middle, 1, 0)); ok {
		t.Fatal("the walls are closed with what's in the chunk that left unknown, and it's a room")
	}
}

func TestARoomOpenedWhilePartlyLeftIsntARoomOnceClosedAgain(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	edge := data.ChunkSize
	start := utils.Vec2i{X: edge, Y: utils.WorldPosToTilePos(testutil.PlayerStart).Y + 20}
	middle := ringSpot(t, s, start, 3, func(middle utils.Vec2i) bool {
		return middle.X == edge && s.Chunks.CanPlaceTile(at(middle, 4, 0))
	})
	wall(t, s, ring(middle, 3)...)
	s.MovePlayer(utils.TilePosToWorldPos(utils.Vec2i{X: (testutil.PlayerLoaderRadius + 2) * data.ChunkSize, Y: middle.Y}))
	s.Tick(5)
	roomAt(t, s, at(middle, 1, 0))

	if destroyed, _ := s.Chunks.DamageTile(at(middle, 3, 0), 1000); !destroyed {
		t.Fatal("the wall couldn't be knocked down")
	}
	if _, ok := s.Chunks.RoomAt(at(middle, 1, 0)); ok {
		t.Fatal("the room is still a room with its wall knocked down")
	}
	wall(t, s, at(middle, 3, 0))
	if _, ok := s.Chunks.RoomAt(at(middle, 1, 0)); ok {
		t.Fatal("closed again, it's a room, though what became of its left half while it was open isn't known")
	}
}
