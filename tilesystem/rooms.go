package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
)

// The tiles of a chunk that air spreads between without going through a wall or door or out of the chunk, which rooms
// are made of. Pieces are numbered by their first tile, so that the same tiles always give the same pieces.
type chunkPieces struct {
	pieceOf *[data.ChunkSize * data.ChunkSize]int32
	pieces  []chunkPiece
}

type chunkPiece struct {
	// Its first tile's index in the chunk
	anchor int32
	tiles  int
}

// What's kept of a chunk with pieces in rooms, so that a room that's partly in chunks that aren't active stays whole
type roomChunk struct {
	pieces []roomPiece
	// For each tile along the chunk's edges, in the order of chunkEdgeIndices, the index of its piece, or edgeWall or
	// edgeOutside
	edges []int32
}

type roomPiece struct {
	anchor int32
	tiles  int
	room   uint64
}

const (
	edgeWall int32 = -1
	// Air in no room, which isn't kept
	edgeOutside int32 = -2
)

type room struct {
	tiles int
}

type roomsDto struct {
	NextId uint64
	Chunks []roomChunkDto
}

type roomChunkDto struct {
	X, Y   int
	Pieces []roomPieceDto
	Edges  []int32
}

type roomPieceDto struct {
	Anchor int32
	Tiles  int
	Room   uint64
}

// For each tile index, its place in chunkEdgeIndices, -1 off the edges
var chunkEdgePos = func() *[data.ChunkSize * data.ChunkSize]int32 {
	pos := &[data.ChunkSize * data.ChunkSize]int32{}
	for i := range pos {
		pos[i] = -1
	}
	for k, i := range chunkEdgeIndices {
		pos[i] = int32(k)
	}
	return pos
}()

func (cc *ChunkContainer) piecesOf(c *chunk) *chunkPieces {
	if c.pieces != nil {
		return c.pieces
	}
	var tight [data.ChunkSize * data.ChunkSize]bool
	// Air spreads over everything but walls and doors, open or not, since an open door is where air flows between rooms
	for i, tile := range c.tiles[data.TileLayer_Wall] {
		tight[i] = cc.isWall(tile)
	}
	for t, f := range c.features {
		if cc.featureAtlas[f].door != nil {
			tight[c.tileIndex(t)] = true
		}
	}
	p := &chunkPieces{pieceOf: &[data.ChunkSize * data.ChunkSize]int32{}}
	for i := range p.pieceOf {
		p.pieceOf[i] = -1
	}
	queue := []int{}
	for start := range p.pieceOf {
		if p.pieceOf[start] != -1 || tight[start] {
			continue
		}
		id := int32(len(p.pieces))
		piece := chunkPiece{anchor: int32(start)}
		p.pieceOf[start] = id
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			piece.tiles++
			x, y := i%data.ChunkSize, i/data.ChunkSize
			for _, j := range [4]int{i - 1, i + 1, i - data.ChunkSize, i + data.ChunkSize} {
				switch {
				case j == i-1 && x == 0, j == i+1 && x == data.ChunkSize-1,
					j == i-data.ChunkSize && y == 0, j == i+data.ChunkSize && y == data.ChunkSize-1:
					continue
				}
				if p.pieceOf[j] == -1 && !tight[j] {
					p.pieceOf[j] = id
					queue = append(queue, j)
				}
			}
		}
		p.pieces = append(p.pieces, piece)
	}
	c.pieces = p
	return p
}

// After a wall or door comes or goes. What the pieces were is kept until the rooms are worked out again, so that the
// rooms they were in carry on in what they've become.
func (cc *ChunkContainer) forgetPieces(c *chunk) {
	if c.pieceRooms != nil {
		c.prevPieces, c.prevPieceRooms = c.pieces, c.pieceRooms
	}
	c.pieces, c.pieceRooms = nil, nil
	cc.roomsStale = true
}

// The rooms, worked out again if a wall or door came or went, or a chunk came or went, since they were last
func (cc *ChunkContainer) updateRooms() {
	if !cc.roomsStale {
		return
	}
	cc.roomsStale = false

	type node struct {
		tiles int
		// How many of its tiles were in each room before
		was map[uint64]int
	}
	nodes := []node{}
	first := map[utils.Vec2i]int{}
	loaded := slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos)
	for _, pos := range loaded {
		c := cc.chunks[pos]
		p := cc.piecesOf(c)
		first[pos] = len(nodes)
		for _, piece := range p.pieces {
			nodes = append(nodes, node{tiles: piece.tiles, was: map[uint64]int{}})
		}
		switch {
		case c.pieceRooms != nil:
			for local, r := range c.pieceRooms {
				if r != 0 {
					nodes[first[pos]+local].was[r] = p.pieces[local].tiles
				}
			}
		case c.prevPieces != nil:
			for i, local := range p.pieceOf {
				if prev := c.prevPieces.pieceOf[i]; local != -1 && prev != -1 && c.prevPieceRooms != nil && c.prevPieceRooms[prev] != 0 {
					nodes[first[pos]+int(local)].was[c.prevPieceRooms[prev]]++
				}
			}
		default:
			if kept, ok := cc.roomChunks[pos]; ok {
				for local, piece := range p.pieces {
					for _, k := range kept.pieces {
						if k.anchor == piece.anchor {
							nodes[first[pos]+local].was[k.room] = piece.tiles
						}
					}
				}
			}
		}
	}
	unloaded := []utils.Vec2i{}
	for _, pos := range slices.SortedFunc(maps.Keys(cc.roomChunks), compareChunkPos) {
		if _, ok := cc.chunks[pos]; ok {
			continue
		}
		unloaded = append(unloaded, pos)
		first[pos] = len(nodes)
		for _, k := range cc.roomChunks[pos].pieces {
			nodes = append(nodes, node{tiles: k.tiles, was: map[uint64]int{k.room: k.tiles}})
		}
	}

	edgeOf := func(pos utils.Vec2i, i int) (int32, bool) {
		if c, ok := cc.chunks[pos]; ok {
			if local := cc.piecesOf(c).pieceOf[i]; local != -1 {
				return local, true
			}
			return edgeWall, true
		}
		if kept, ok := cc.roomChunks[pos]; ok {
			return kept.edges[chunkEdgePos[i]], true
		}
		return 0, false
	}

	parent := make([]int, len(nodes))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	open := make([]bool, len(nodes))
	for _, pos := range append(slices.Clone(loaded), unloaded...) {
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		for _, i := range chunkEdgeIndices {
			local, _ := edgeOf(pos, i)
			if local < 0 {
				continue
			}
			id := first[pos] + int(local)
			t := utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize}
			for _, n := range edgeNeighbours(t) {
				npos := utils.Vec2i{X: floorDiv(n.X, data.ChunkSize), Y: floorDiv(n.Y, data.ChunkSize)}
				if npos == pos {
					continue
				}
				nLocal, known := edgeOf(npos, (n.Y-npos.Y*data.ChunkSize)*data.ChunkSize+n.X-npos.X*data.ChunkSize)
				switch {
				case !known || nLocal == edgeOutside:
					open[id] = true
				case nLocal != edgeWall:
					parent[find(id)] = find(first[npos] + int(nLocal))
				}
			}
		}
	}

	tiles, openRoot, members := map[int]int{}, map[int]bool{}, map[int][]int{}
	roots := []int{}
	for i, n := range nodes {
		r := find(i)
		if _, seen := tiles[r]; !seen {
			roots = append(roots, r)
		}
		tiles[r] += n.tiles
		openRoot[r] = openRoot[r] || open[i]
		members[r] = append(members[r], i)
	}
	isRoom := func(r int) bool { return !openRoot[r] && tiles[r] <= data.RoomMaxTiles }

	// Each room that was carries on in what has most of its tiles now
	type claim struct {
		root, tiles int
		room        uint64
	}
	claims := []claim{}
	for _, r := range roots {
		if !isRoom(r) {
			continue
		}
		was := map[uint64]int{}
		for _, i := range members[r] {
			for room, count := range nodes[i].was {
				was[room] += count
			}
		}
		for room, count := range was {
			claims = append(claims, claim{root: r, tiles: count, room: room})
		}
	}
	slices.SortFunc(claims, func(a, b claim) int {
		return cmp.Or(cmp.Compare(b.tiles, a.tiles), cmp.Compare(a.room, b.room), cmp.Compare(a.root, b.root))
	})
	roomOf := map[int]uint64{}
	taken := map[uint64]bool{}
	for _, c := range claims {
		if _, ok := roomOf[c.root]; !ok && !taken[c.room] {
			roomOf[c.root], taken[c.room] = c.room, true
		}
	}
	cc.rooms = map[uint64]*room{}
	for _, r := range roots {
		if !isRoom(r) {
			continue
		}
		if _, ok := roomOf[r]; !ok {
			cc.nextRoomId++
			roomOf[r] = cc.nextRoomId
		}
		cc.rooms[roomOf[r]] = &room{tiles: tiles[r]}
	}

	for _, pos := range loaded {
		c := cc.chunks[pos]
		p := cc.piecesOf(c)
		rooms := make([]uint64, len(p.pieces))
		for local := range p.pieces {
			rooms[local] = roomOf[find(first[pos]+local)]
		}
		cc.noteEnclosed(pos, c, p, rooms)
		c.pieceRooms, c.prevPieces, c.prevPieceRooms = rooms, nil, nil
	}

	kept := map[utils.Vec2i]*roomChunk{}
	keep := func(pos utils.Vec2i, pieces []roomPiece, edge func(i int) int32) {
		index := map[int32]int32{}
		k := &roomChunk{}
		for local, piece := range pieces {
			if piece.room != 0 {
				index[int32(local)] = int32(len(k.pieces))
				k.pieces = append(k.pieces, piece)
			}
		}
		if len(k.pieces) == 0 {
			return
		}
		k.edges = make([]int32, len(chunkEdgeIndices))
		for e, i := range chunkEdgeIndices {
			local := edge(i)
			switch to, inRoom := index[local]; {
			case local == edgeWall:
				k.edges[e] = edgeWall
			case inRoom:
				k.edges[e] = to
			default:
				k.edges[e] = edgeOutside
			}
		}
		kept[pos] = k
	}
	for _, pos := range loaded {
		c := cc.chunks[pos]
		p := cc.piecesOf(c)
		pieces := make([]roomPiece, len(p.pieces))
		for local, piece := range p.pieces {
			pieces[local] = roomPiece{anchor: piece.anchor, tiles: piece.tiles, room: c.pieceRooms[local]}
		}
		keep(pos, pieces, func(i int) int32 {
			if local := p.pieceOf[i]; local != -1 {
				return local
			}
			return edgeWall
		})
	}
	for _, pos := range unloaded {
		old := cc.roomChunks[pos]
		pieces := make([]roomPiece, len(old.pieces))
		for local, piece := range old.pieces {
			pieces[local] = roomPiece{anchor: piece.anchor, tiles: piece.tiles, room: roomOf[find(first[pos]+local)]}
		}
		keep(pos, pieces, func(i int) int32 { return old.edges[chunkEdgePos[i]] })
	}
	cc.roomChunks = kept
}

// The tiles of the chunk that have just come to be in a room, from in none, for the roof area to be put over
func (cc *ChunkContainer) noteEnclosed(pos utils.Vec2i, c *chunk, p *chunkPieces, rooms []uint64) {
	was := func(i int, local int32) uint64 {
		switch {
		case c.pieceRooms != nil:
			return c.pieceRooms[local]
		case c.prevPieces != nil:
			if prev := c.prevPieces.pieceOf[i]; prev != -1 && c.prevPieceRooms != nil {
				return c.prevPieceRooms[prev]
			}
			return 0
		}
		if kept, ok := cc.roomChunks[pos]; ok {
			for _, k := range kept.pieces {
				if k.anchor == p.pieces[local].anchor {
					return k.room
				}
			}
		}
		return 0
	}
	origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
	for i, local := range p.pieceOf {
		if local != -1 && rooms[local] != 0 && was(i, local) == 0 {
			cc.enclosed = append(cc.enclosed, enclosedTile{tile: utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize}, room: rooms[local]})
		}
	}
}

type enclosedTile struct {
	tile utils.Vec2i
	room uint64
}

// Works out the rooms if anything's changed, and puts the roof area over the tiles of rooms on the home area that have
// just been enclosed, and the walls and doors round them, other than where the no-roof area is
func (cc *ChunkContainer) TickRooms(x *ecs.ECSContainer) {
	cc.updateRooms()
	if len(cc.enclosed) == 0 {
		return
	}
	enclosed := cc.enclosed
	cc.enclosed = nil
	onHome := map[uint64]bool{}
	for _, e := range enclosed {
		if _, ok := onHome[e.room]; !ok {
			onHome[e.room] = cc.roomOnHome(e.room)
		}
	}
	for _, e := range enclosed {
		if !onHome[e.room] {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				t := utils.Vec2i{X: e.tile.X + dx, Y: e.tile.Y + dy}
				if (t == e.tile || cc.isAirTight(t)) && !cc.InArea(data.Area_NoRoof, t) {
					x.MarkArea(data.Area_Roof, t, t, true)
				}
			}
		}
	}
}

func (cc *ChunkContainer) isAirTight(t utils.Vec2i) bool {
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	return ok && cc.piecesOf(c).pieceOf[c.tileIndex(t)] == -1
}

func (cc *ChunkContainer) roomOnHome(id uint64) bool {
	found := false
	cc.roomTiles(id, func(t utils.Vec2i) bool {
		found = cc.InArea(data.Area_Home, t)
		return !found
	})
	return found
}

// Calls each with the room's active tiles, row by row within each chunk, until it returns false
func (cc *ChunkContainer) roomTiles(id uint64, each func(t utils.Vec2i) bool) {
	for _, pos := range slices.SortedFunc(maps.Keys(cc.chunks), compareChunkPos) {
		c := cc.chunks[pos]
		if c.pieceRooms == nil || !slices.Contains(c.pieceRooms, id) {
			continue
		}
		origin := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		for i, local := range c.pieces.pieceOf {
			if local != -1 && c.pieceRooms[local] == id && !each(utils.Vec2i{X: origin.X + i%data.ChunkSize, Y: origin.Y + i/data.ChunkSize}) {
				return
			}
		}
	}
}

// The room the tile is in, false outdoors, on a wall or door, or where nothing is active
func (cc *ChunkContainer) RoomAt(t utils.Vec2i) (uint64, bool) {
	cc.updateRooms()
	c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
	if !ok {
		return 0, false
	}
	local := cc.piecesOf(c).pieceOf[c.tileIndex(t)]
	if local == -1 || c.pieceRooms[local] == 0 {
		return 0, false
	}
	return c.pieceRooms[local], true
}

// How many tiles the room has, active or not
func (cc *ChunkContainer) RoomSize(id uint64) int {
	cc.updateRooms()
	if r, ok := cc.rooms[id]; ok {
		return r.tiles
	}
	return 0
}

// The room the tile is in, how big it is and how much of it is open to the sky, for the debug tooltip
func (cc *ChunkContainer) RoomDebugLines(t utils.Vec2i) []string {
	if cc.isAirTight(t) {
		return nil
	}
	id, ok := cc.RoomAt(t)
	if !ok {
		if _, active := cc.TileAt(t); active {
			return []string{"Outdoors"}
		}
		return nil
	}
	unroofed := 0
	cc.roomTiles(id, func(t utils.Vec2i) bool {
		if cc.IsOpenToSky(t) {
			unroofed++
		}
		return true
	})
	return []string{fmt.Sprintf("Room %d: %d tiles, %d unroofed", id, cc.RoomSize(id), unroofed)}
}

func (cc *ChunkContainer) roomsToDto() roomsDto {
	cc.updateRooms()
	dto := roomsDto{NextId: cc.nextRoomId}
	for _, pos := range slices.SortedFunc(maps.Keys(cc.roomChunks), compareChunkPos) {
		k := cc.roomChunks[pos]
		d := roomChunkDto{X: pos.X, Y: pos.Y, Edges: slices.Clone(k.edges)}
		for _, p := range k.pieces {
			d.Pieces = append(d.Pieces, roomPieceDto{Anchor: p.anchor, Tiles: p.tiles, Room: p.room})
		}
		dto.Chunks = append(dto.Chunks, d)
	}
	return dto
}

// Reads the rooms as of the last commit, with none for a world that has never had any
func (cc *ChunkContainer) LoadRooms() error {
	file, err := os.Open(data.RoomsFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error opening rooms file: %w", err)
	}
	defer file.Close()
	var dto roomsDto
	if err := gob.NewDecoder(file).Decode(&dto); err != nil {
		return fmt.Errorf("error decoding rooms file: %w", err)
	}
	cc.nextRoomId = dto.NextId
	cc.rooms = map[uint64]*room{}
	cc.roomChunks = map[utils.Vec2i]*roomChunk{}
	for _, d := range dto.Chunks {
		if len(d.Edges) != len(chunkEdgeIndices) {
			return fmt.Errorf("rooms file has %d edge tiles for the chunk at %d,%d, want %d", len(d.Edges), d.X, d.Y, len(chunkEdgeIndices))
		}
		k := &roomChunk{edges: d.Edges}
		for _, p := range d.Pieces {
			k.pieces = append(k.pieces, roomPiece{anchor: p.Anchor, tiles: p.Tiles, room: p.Room})
			if cc.rooms[p.Room] == nil {
				cc.rooms[p.Room] = &room{}
			}
			cc.rooms[p.Room].tiles += p.Tiles
		}
		cc.roomChunks[utils.Vec2i{X: d.X, Y: d.Y}] = k
	}
	cc.roomsStale = true
	return nil
}
