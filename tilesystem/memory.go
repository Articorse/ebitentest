package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// What the player remembers of the terrain and the features on it, which can differ from what's there now. It's kept in regions that are
// all held in memory, since remembered ground has to be drawn far beyond the active chunks.
type memory struct {
	regions map[utils.Vec2i]*memoryRegion
	version uint64
}

type memoryRegion struct {
	// Tile_None where nothing is remembered
	tiles [data.MemoryRegionSize * data.MemoryRegionSize]data.TileEnum
	// Feature_None where the tile had none, or isn't remembered
	features [data.MemoryRegionSize * data.MemoryRegionSize]data.FeatureEnum
	version  uint64
	dirty    bool
}

type memoryRegionDto struct {
	X, Y     int
	Tiles    []data.TileEnum
	Features []data.FeatureEnum
}

func newMemory() memory {
	return memory{regions: make(map[utils.Vec2i]*memoryRegion)}
}

func (m *memory) remember(t utils.Vec2i, tile data.TileEnum, feature data.FeatureEnum) {
	pos, idx := memoryIndex(t)
	r, ok := m.regions[pos]
	if !ok {
		r = &memoryRegion{}
		m.regions[pos] = r
	}

	if r.tiles[idx] == tile && r.features[idx] == feature {
		return
	}

	m.version++
	r.tiles[idx] = tile
	r.features[idx] = feature
	r.version = m.version
	r.dirty = true
}

func (m *memory) recall(t utils.Vec2i) (data.TileEnum, bool) {
	pos, idx := memoryIndex(t)
	r, ok := m.regions[pos]
	if !ok || r.tiles[idx] == data.Tile_None {
		return data.Tile_None, false
	}
	return r.tiles[idx], true
}

func (m *memory) recallFeature(t utils.Vec2i) (data.FeatureEnum, bool) {
	pos, idx := memoryIndex(t)
	r, ok := m.regions[pos]
	if !ok || r.features[idx] == data.Feature_None {
		return data.Feature_None, false
	}
	return r.features[idx], true
}

func memoryIndex(t utils.Vec2i) (region utils.Vec2i, idx int) {
	region = utils.Vec2i{X: floorDiv(t.X, data.MemoryRegionSize), Y: floorDiv(t.Y, data.MemoryRegionSize)}
	localX := t.X - region.X*data.MemoryRegionSize
	localY := t.Y - region.Y*data.MemoryRegionSize
	return region, localY*data.MemoryRegionSize + localX
}

func (cc *ChunkContainer) RecallTile(t utils.Vec2i) (data.TileEnum, bool) {
	return cc.memory.recall(t)
}

func (cc *ChunkContainer) RecallFeature(t utils.Vec2i) (data.FeatureEnum, bool) {
	return cc.memory.recallFeature(t)
}

// The sprites of the remembered features whose tiles are within the bounds, in tile order
func (cc *ChunkContainer) RememberedFeatureSprites(bounds [2]utils.Vec2f) []common.StaticSprite {
	minTile, maxTile := utils.WorldPosToTilePos(bounds[0]), utils.WorldPosToTilePos(bounds[1])
	sprites := []common.StaticSprite{}
	for y := minTile.Y; y <= maxTile.Y; y++ {
		for x := minTile.X; x <= maxTile.X; x++ {
			t := utils.Vec2i{X: x, Y: y}
			f, ok := cc.memory.recallFeature(t)
			if !ok {
				continue
			}
			if sprite, ok := cc.featureSprite(t, f); ok {
				sprites = append(sprites, sprite)
			}
		}
	}
	return sprites
}

// Remembers the tiles, and the features on them, that the viewer's vision circle shows completely, so that remembered
// ground never reaches past where the circle has been
func (cc *ChunkContainer) RememberSeenTiles(viewerPos utils.Vec2f, vision *ecs.Vision) {
	clearRadius, _ := vision.CircleRadii()
	center := utils.WorldPosToTilePos(viewerPos)
	reach := int(math.Ceil(clearRadius/data.TileSize)) + 1

	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			t := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
			if !vision.CanSee(t) || !tileInsideCircle(t, viewerPos, clearRadius) {
				continue
			}

			c, ok := cc.chunks[utils.Vec2i{X: floorDiv(t.X, data.ChunkSize), Y: floorDiv(t.Y, data.ChunkSize)}]
			if !ok {
				continue
			}
			cc.memory.remember(t, c.getTileDefId(t), c.features[t])
		}
	}
}

func tileInsideCircle(t utils.Vec2i, center utils.Vec2f, radius float64) bool {
	c := utils.TilePosToWorldPos(t)
	farX := math.Abs(c.X-center.X) + data.TileSize/2
	farY := math.Abs(c.Y-center.Y) + data.TileSize/2
	return farX*farX+farY*farY <= radius*radius
}

// Draws every remembered tile on screen darkened, and the rest black
func (cc *ChunkContainer) DrawMemory(dst *ebiten.Image, camera utils.Vec2f) {
	dst.Fill(color.Black)

	size := dst.Bounds().Size()
	minTile := utils.WorldPosToTilePos(camera)
	maxTile := utils.WorldPosToTilePos(camera.Add(utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}))

	for y := minTile.Y; y <= maxTile.Y; y++ {
		for x := minTile.X; x <= maxTile.X; x++ {
			t := utils.Vec2i{X: x, Y: y}
			tile, ok := cc.memory.recall(t)
			if !ok {
				continue
			}
			def, ok := cc.atlas[tile]
			if !ok {
				continue
			}

			pos := utils.TilePosToWorldPos(t)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(pos.X-data.TileSize/2-camera.X, pos.Y-data.TileSize/2-camera.Y)
			op.ColorScale.Scale(data.MemoryBrightness, data.MemoryBrightness, data.MemoryBrightness, 1)
			dst.DrawImage(def.image, op)
		}
	}
}

// Reads every saved region, so must only run once a world's files are recovered
func (cc *ChunkContainer) LoadMemory() error {
	entries, err := os.ReadDir(data.ChunkDataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error reading chunk data directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, data.MemoryFilePrefix) || filepath.Ext(name) != ".gob" {
			continue
		}

		dto, err := readMemoryFile(filepath.Join(data.ChunkDataDir, name))
		if err != nil {
			return err
		}
		if n := data.MemoryRegionSize * data.MemoryRegionSize; len(dto.Tiles) != n || len(dto.Features) != n {
			return fmt.Errorf("memory file %s has %d tiles and %d features, want %d of each", name, len(dto.Tiles), len(dto.Features), n)
		}

		r := &memoryRegion{}
		copy(r.tiles[:], dto.Tiles)
		copy(r.features[:], dto.Features)
		cc.memory.regions[utils.Vec2i{X: dto.X, Y: dto.Y}] = r
	}

	return nil
}

func readMemoryFile(p string) (dto memoryRegionDto, err error) {
	file, err := os.Open(p)
	if err != nil {
		return dto, fmt.Errorf("error opening memory file %s: %w", p, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close file: %w", cerr))
		}
	}()

	if err := gob.NewDecoder(file).Decode(&dto); err != nil {
		return dto, fmt.Errorf("error decoding memory file %s: %w", p, err)
	}
	return dto, nil
}

func memoryTmpFilePath(pos utils.Vec2i, generation uint64) string {
	return fmt.Sprintf(data.MemoryFilePathTemplate+".%d.tmp", pos.X, pos.Y, generation)
}
