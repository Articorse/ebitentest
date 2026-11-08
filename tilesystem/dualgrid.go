package tilesystem

import (
	"cmp"
	"ebittest/data"
	"ebittest/utils"
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// Where on a dual-grid sheet the cell for each corner sits, by which of the four tiles around the corner are the
// sheet's: the top-left one adds 1, top-right 2, bottom-left 4 and bottom-right 8
var dualGridCells = [16]int{0: 12, 1: 15, 2: 8, 3: 9, 4: 0, 5: 11, 6: 14, 7: 7, 8: 13, 9: 4, 10: 1, 11: 10, 12: 3, 13: 2, 14: 5, 15: 6}

// One of what's drawn on a corner: the cell of a tile's dual-grid sheet, or a quarter of the image of a tile drawn whole
type CornerPiece struct {
	Tile data.TileEnum
	// Of the tile's sheet, or -1 for a quarter of its image
	Cell int
	// For a quarter, which of the four tiles around the corner it's on: top-left 0, top-right 1, bottom-left 2 and
	// bottom-right 3
	Quarter int
}

// The tiles are drawn on a grid half a tile off from theirs, where each cell sits on the corner four tiles meet at and
// is drawn from what those four are, so that where tiles meet is drawn as a whole rather than as the edges of each. The
// cell's quarters lie on the four tiles. The layers are drawn from the bottom up, so that what doesn't cover a whole
// tile shows what's under it. Within a layer, a tile drawn whole has the quarter of its image there drawn under the
// rest, and the sheets are drawn over each other by draw order. False while one of the four isn't known, which isn't
// drawn, so that the edge of what's loaded or remembered isn't drawn as though it were the edge of the world.
func (cc *ChunkContainer) CornerPieces(corner utils.Vec2i, tileAt func(utils.Vec2i) (data.TileLayers, bool)) ([]CornerPiece, bool) {
	var tiles [4]data.TileLayers
	for i := range tiles {
		layers, ok := tileAt(utils.Vec2i{X: corner.X - 1 + i%2, Y: corner.Y - 1 + i/2})
		if !ok {
			return nil, false
		}
		tiles[i] = layers
	}

	pieces := []CornerPiece{}
	for layer := range data.TileLayerCount {
		sheets := []data.TileEnum{}
		for i, layers := range tiles {
			tile := layers[layer]
			if tile == data.Tile_None {
				continue
			}
			switch def := cc.atlas[tile]; {
			case def.dualGrid != nil:
				if !slices.Contains(sheets, tile) {
					sheets = append(sheets, tile)
				}
			case def.image != nil:
				pieces = append(pieces, CornerPiece{Tile: tile, Cell: -1, Quarter: i})
			}
		}
		slices.SortFunc(sheets, func(a, b data.TileEnum) int {
			return cmp.Or(cmp.Compare(cc.atlas[a].drawOrder, cc.atlas[b].drawOrder), cmp.Compare(a, b))
		})
		for _, sheet := range sheets {
			mask := 0
			for i, layers := range tiles {
				if layers[layer] == sheet {
					mask |= 1 << i
				}
			}
			pieces = append(pieces, CornerPiece{Tile: sheet, Cell: dualGridCells[mask]})
		}
	}
	return pieces, true
}

// topLeft is the tile whose top-left corner is the image's
func (cc *ChunkContainer) drawCorner(dst *ebiten.Image, topLeft, corner utils.Vec2i, tileAt func(utils.Vec2i) (data.TileLayers, bool)) {
	pieces, ok := cc.CornerPieces(corner, tileAt)
	if !ok {
		return
	}
	at := utils.Vec2f{
		X: float64((corner.X-topLeft.X)*data.TileSize - data.TileSize/2),
		Y: float64((corner.Y-topLeft.Y)*data.TileSize - data.TileSize/2),
	}
	half := data.TileSize / 2
	for _, p := range pieces {
		def := cc.atlas[p.Tile]
		op := &ebiten.DrawImageOptions{}
		if p.Cell != -1 {
			op.GeoM.Translate(at.X, at.Y)
			dst.DrawImage(def.dualGrid[p.Cell], op)
			continue
		}
		// The quarter of the tile at the corner, which is the one opposite where the tile lies from it
		x, y := (1-p.Quarter%2)*half, (1-p.Quarter/2)*half
		op.GeoM.Translate(at.X+float64(p.Quarter%2*half), at.Y+float64(p.Quarter/2*half))
		dst.DrawImage(def.image.SubImage(image.Rect(x, y, x+half, y+half)).(*ebiten.Image), op)
	}
}

// Every corner of the tiles from topLeft on that the image covers, the image being cleared first
func (cc *ChunkContainer) drawTiles(dst *ebiten.Image, topLeft utils.Vec2i, side int, tileAt func(utils.Vec2i) (data.TileLayers, bool)) {
	dst.Clear()
	for y := topLeft.Y; y <= topLeft.Y+side; y++ {
		for x := topLeft.X; x <= topLeft.X+side; x++ {
			cc.drawCorner(dst, topLeft, utils.Vec2i{X: x, Y: y}, tileAt)
		}
	}
}

// Draws the corner again in every chunk image it lies in, once a tile around it has changed
func (cc *ChunkContainer) redrawCorner(corner utils.Vec2i) {
	done := []utils.Vec2i{}
	for i := range 4 {
		t := utils.Vec2i{X: corner.X - 1 + i%2, Y: corner.Y - 1 + i/2}
		pos := utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}
		c, ok := cc.chunks[pos]
		if !ok || c.Image == nil || slices.Contains(done, pos) {
			continue
		}
		done = append(done, pos)

		topLeft := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
		px, py := (corner.X-topLeft.X)*data.TileSize, (corner.Y-topLeft.Y)*data.TileSize
		half := data.TileSize / 2
		c.Image.SubImage(image.Rect(px-half, py-half, px+half, py+half)).(*ebiten.Image).Clear()
		cc.drawCorner(c.Image, topLeft, corner, cc.TileAt)
	}
}

// The corners around the tile
func (cc *ChunkContainer) redrawAround(t utils.Vec2i) {
	for i := range 4 {
		cc.redrawCorner(utils.Vec2i{X: t.X + i%2, Y: t.Y + i/2})
	}
}

// The corners along the chunk's edges, which the chunks around it draw too, but couldn't while it wasn't there
func (cc *ChunkContainer) redrawEdges(pos utils.Vec2i) {
	topLeft := utils.Vec2i{X: pos.X * data.ChunkSize, Y: pos.Y * data.ChunkSize}
	n := data.ChunkSize
	for i := 0; i <= n; i++ {
		for _, corner := range []utils.Vec2i{
			{X: topLeft.X + i, Y: topLeft.Y},
			{X: topLeft.X + i, Y: topLeft.Y + n},
			{X: topLeft.X, Y: topLeft.Y + i},
			{X: topLeft.X + n, Y: topLeft.Y + i},
		} {
			cc.redrawCorner(corner)
		}
	}
}
