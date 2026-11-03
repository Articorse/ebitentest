package tilesystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"image"
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

	// Nil until the region comes into view, and again once it's far enough out of it
	image      *ebiten.Image
	imageStale bool
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

	featureChanged := r.features[idx] != feature
	m.version++
	r.tiles[idx] = tile
	r.features[idx] = feature
	r.version = m.version
	r.dirty = true
	r.imageStale = true

	if featureChanged {
		m.markFeatureStale(t)
	}
}

// A feature's sprite reaches past its tile, so it's baked into every region it may reach into
func (m *memory) markFeatureStale(t utils.Vec2i) {
	reach := data.MemoryFeatureReachTiles
	minRegion, _ := memoryIndex(utils.Vec2i{X: t.X - reach, Y: t.Y - reach})
	maxRegion, _ := memoryIndex(utils.Vec2i{X: t.X + reach, Y: t.Y + reach})
	for y := minRegion.Y; y <= maxRegion.Y; y++ {
		for x := minRegion.X; x <= maxRegion.X; x++ {
			if r, ok := m.regions[utils.Vec2i{X: x, Y: y}]; ok {
				r.imageStale = true
			}
		}
	}
}

func memoryRegionBounds(pos utils.Vec2i) [2]utils.Vec2f {
	side := float64(data.MemoryRegionSize * data.TileSize)
	topLeft := utils.Vec2f{X: float64(pos.X) * side, Y: float64(pos.Y) * side}.
		Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})
	return [2]utils.Vec2f{topLeft, topLeft.Add(utils.Vec2f{X: side, Y: side})}
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

// Draws what's remembered on screen darkened, and the rest black. Each region is baked into an image, which is kept
// while it's near the screen and baked again only once something in it is remembered differently, so that drawing
// memory takes a few draws however much is remembered. Remembered features are drawn into the images by drawFeatures.
func (cc *ChunkContainer) DrawMemory(
	dst *ebiten.Image,
	camera utils.Vec2f,
	drawFeatures func(dst *ebiten.Image, camera utils.Vec2f, sprites []common.StaticSprite),
) {
	dst.Fill(color.Black)

	size := dst.Bounds().Size()
	view := utils.Vec2f{X: float64(size.X), Y: float64(size.Y)}
	keep := cameraBounds(camera, view, data.MemoryImageFreeMarginPx)
	for pos, r := range cc.memory.regions {
		if r.image != nil && !utils.DetectAABBCollision(memoryRegionBounds(pos), keep) {
			r.image.Deallocate()
			r.image = nil
		}
	}

	minRegion, _ := memoryIndex(utils.WorldPosToTilePos(camera))
	maxRegion, _ := memoryIndex(utils.WorldPosToTilePos(camera.Add(view)))
	bakesLeft := data.MaxMemoryBakesPerFrame

	for y := minRegion.Y; y <= maxRegion.Y; y++ {
		for x := minRegion.X; x <= maxRegion.X; x++ {
			pos := utils.Vec2i{X: x, Y: y}
			r, ok := cc.memory.regions[pos]
			if !ok {
				continue
			}

			if (r.image == nil || r.imageStale) && bakesLeft > 0 {
				cc.bakeMemoryRegion(pos, r, drawFeatures)
				bakesLeft--
			}
			// A stale image is still what was remembered until shortly before, so it's drawn until it's baked again
			if r.image == nil {
				continue
			}

			topLeft := memoryRegionBounds(pos)[0]
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(topLeft.X-camera.X, topLeft.Y-camera.Y)
			dst.DrawImage(r.image, op)
		}
	}
}

func (cc *ChunkContainer) bakeMemoryRegion(
	pos utils.Vec2i,
	r *memoryRegion,
	drawFeatures func(dst *ebiten.Image, camera utils.Vec2f, sprites []common.StaticSprite),
) {
	side := data.MemoryRegionSize * data.TileSize
	if r.image == nil {
		// Unmanaged like chunk images, so that region images don't grow a page of the shared atlas
		r.image = ebiten.NewImageWithOptions(image.Rect(0, 0, side, side), &ebiten.NewImageOptions{Unmanaged: true})
	} else {
		r.image.Clear()
	}

	for i, tile := range r.tiles {
		def, ok := cc.atlas[tile]
		if tile == data.Tile_None || !ok {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i%data.MemoryRegionSize*data.TileSize), float64(i/data.MemoryRegionSize*data.TileSize))
		op.ColorScale.Scale(data.MemoryBrightness, data.MemoryBrightness, data.MemoryBrightness, 1)
		r.image.DrawImage(def.image, op)
	}

	// Including the features of the regions around it whose sprites reach into it, so that they're drawn in order with
	// its own
	bounds := memoryRegionBounds(pos)
	reach := utils.Vec2f{X: data.MemoryFeatureReachTiles * data.TileSize, Y: data.MemoryFeatureReachTiles * data.TileSize}
	drawFeatures(r.image, bounds[0], cc.RememberedFeatureSprites([2]utils.Vec2f{bounds[0].Subtract(reach), bounds[1].Add(reach)}))

	r.imageStale = false
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
