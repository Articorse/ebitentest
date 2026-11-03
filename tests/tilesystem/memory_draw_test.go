package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"math"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

const regionPx = data.MemoryRegionSize * data.TileSize

// Records which regions are baked, by the top left of their images, and the remembered features drawn into each
type bakes struct {
	regions  []utils.Vec2i
	features [][]common.StaticSprite
}

func (b *bakes) drawFeatures(_ *ebiten.Image, camera utils.Vec2f, sprites []common.StaticSprite) {
	b.regions = append(b.regions, regionAt(camera.Add(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2})))
	b.features = append(b.features, sprites)
}

func (b *bakes) drawn(s *testutil.Session, dst *ebiten.Image, camera utils.Vec2f) []utils.Vec2i {
	before := len(b.regions)
	s.Chunks.DrawMemory(dst, camera, b.drawFeatures)
	return b.regions[before:]
}

func regionAt(pos utils.Vec2f) utils.Vec2i {
	t := utils.WorldPosToTilePos(pos)
	return utils.Vec2i{
		X: int(math.Floor(float64(t.X) / data.MemoryRegionSize)),
		Y: int(math.Floor(float64(t.Y) / data.MemoryRegionSize)),
	}
}

// The regions in view that hold something remembered
func rememberedRegionsInView(s *testutil.Session, camera utils.Vec2f, size utils.Vec2f) []utils.Vec2i {
	regions := []utils.Vec2i{}
	minTile, maxTile := utils.WorldPosToTilePos(camera), utils.WorldPosToTilePos(camera.Add(size))
	for y := minTile.Y; y <= maxTile.Y; y++ {
		for x := minTile.X; x <= maxTile.X; x++ {
			t := utils.Vec2i{X: x, Y: y}
			if _, ok := s.Chunks.RecallTile(t); ok {
				r := regionAt(utils.TilePosToWorldPos(t))
				if !slices.Contains(regions, r) {
					regions = append(regions, r)
				}
			}
		}
	}
	return regions
}

func remembersAnything(s *testutil.Session, region utils.Vec2i) bool {
	for y := range data.MemoryRegionSize {
		for x := range data.MemoryRegionSize {
			if _, ok := s.Chunks.RecallTile(utils.Vec2i{X: region.X*data.MemoryRegionSize + x, Y: region.Y*data.MemoryRegionSize + y}); ok {
				return true
			}
		}
	}
	return false
}

func sameRegions(a, b []utils.Vec2i) bool {
	sort := func(r []utils.Vec2i) []utils.Vec2i {
		r = slices.Clone(r)
		slices.SortFunc(r, func(p, q utils.Vec2i) int {
			if p.Y != q.Y {
				return p.Y - q.Y
			}
			return p.X - q.X
		})
		return r
	}
	return slices.Equal(sort(a), sort(b))
}

func TestMemoryRegionsAreBakedOnceUntilSomethingInThemChanges(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	see(s, testutil.PlayerStart)
	dst := ebiten.NewImage(640, 360)
	camera := testutil.PlayerStart.Subtract(utils.Vec2f{X: 320, Y: 180})
	b := &bakes{}

	want := rememberedRegionsInView(s, camera, utils.Vec2f{X: 640, Y: 360})
	if len(want) == 0 || len(want) > data.MaxMemoryBakesPerFrame {
		t.Fatalf("%d regions in view, want some, but no more than are baked in a frame", len(want))
	}
	if got := b.drawn(s, dst, camera); !sameRegions(got, want) {
		t.Fatalf("baked regions %v, want the remembered ones in view %v", got, want)
	}
	if got := b.drawn(s, dst, camera); len(got) != 0 {
		t.Fatalf("regions %v were baked again without anything new remembered", got)
	}

	see(s, testutil.PlayerStart)
	if got := b.drawn(s, dst, camera); len(got) != 0 {
		t.Fatalf("regions %v were baked again after seeing the same again", got)
	}

	remembered := func() map[utils.Vec2i]bool {
		tiles := map[utils.Vec2i]bool{}
		center := utils.WorldPosToTilePos(testutil.PlayerStart)
		for dy := -20; dy <= 20; dy++ {
			for dx := -20; dx <= 20; dx++ {
				tile := utils.Vec2i{X: center.X + dx, Y: center.Y + dy}
				if _, ok := s.Chunks.RecallTile(tile); ok {
					tiles[tile] = true
				}
			}
		}
		return tiles
	}
	before := remembered()
	see(s, testutil.PlayerStart.Add(utils.Vec2f{X: 3 * data.TileSize, Y: -2 * data.TileSize}))

	changed := []utils.Vec2i{}
	for tile := range remembered() {
		if r := regionAt(utils.TilePosToWorldPos(tile)); !before[tile] && !slices.Contains(changed, r) {
			changed = append(changed, r)
		}
	}
	if len(changed) == 0 {
		t.Fatal("seeing from elsewhere didn't remember anything new")
	}
	if got := b.drawn(s, dst, camera); !sameRegions(got, changed) {
		t.Fatalf("baked regions %v after something new was remembered, want the regions it's in %v", got, changed)
	}
}

func TestOnlyAFewMemoryRegionsAreBakedPerFrame(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	for i := range data.MaxMemoryBakesPerFrame + 2 {
		viewer := testutil.PlayerStart.Add(utils.Vec2f{X: float64(i * regionPx)})
		s.MovePlayer(viewer)
		s.Tick(5)
		see(s, viewer)
	}

	size := utils.Vec2f{X: float64((data.MaxMemoryBakesPerFrame + 3) * regionPx), Y: 360}
	dst := ebiten.NewImage(int(size.X), int(size.Y))
	camera := testutil.PlayerStart.Subtract(utils.Vec2f{X: regionPx, Y: 180})
	want := rememberedRegionsInView(s, camera, size)
	if len(want) <= data.MaxMemoryBakesPerFrame {
		t.Fatalf("only %d regions remembered in view, want more than are baked in a frame", len(want))
	}

	b := &bakes{}
	if got := b.drawn(s, dst, camera); len(got) != data.MaxMemoryBakesPerFrame {
		t.Fatalf("%d regions baked in the first frame, want %d", len(got), data.MaxMemoryBakesPerFrame)
	}
	for range len(want) {
		b.drawn(s, dst, camera)
	}
	if !sameRegions(b.regions, want) {
		t.Fatalf("baked regions %v over several frames, want each in view once %v", b.regions, want)
	}
}

func TestMemoryRegionsFarOffScreenAreFreedAndBakedAgain(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	see(s, testutil.PlayerStart)
	dst := ebiten.NewImage(640, 360)
	camera := testutil.PlayerStart.Subtract(utils.Vec2f{X: 320, Y: 180})
	b := &bakes{}
	first := b.drawn(s, dst, camera)

	nearby := camera.Add(utils.Vec2f{X: 640 + data.MemoryImageFreeMarginPx/2})
	b.drawn(s, dst, nearby)
	if got := b.drawn(s, dst, camera); len(got) != 0 {
		t.Fatalf("regions %v were baked again after looking just past them", got)
	}

	b.drawn(s, dst, camera.Add(utils.Vec2f{X: 50 * regionPx}))
	if got := b.drawn(s, dst, camera); !sameRegions(got, first) {
		t.Fatalf("regions %v were baked after coming back from far away, want them all again %v", got, first)
	}
}

func TestRememberingAFeatureBakesTheRegionsItsSpriteReaches(t *testing.T) {
	testutil.TempWorld(t)
	s := testutil.StartSession(t)
	tree := findLoneTree(t, s)
	treePos := utils.TilePosToWorldPos(tree)
	see(s, treePos)

	size := utils.Vec2f{X: 3 * regionPx, Y: 3 * regionPx}
	dst := ebiten.NewImage(int(size.X), int(size.Y))
	camera := treePos.Subtract(size.Multiply(0.5))
	b := &bakes{}
	for range 4 {
		b.drawn(s, dst, camera)
	}

	if !slices.Contains(b.regions, regionAt(treePos)) {
		t.Fatalf("the tree's region %v wasn't baked", regionAt(treePos))
	}
	// Every remembered feature whose sprite may reach into a region is drawn into it, its neighbors' included
	reach := data.MemoryFeatureReachTiles
	for i, sprites := range b.features {
		region := b.regions[i]
		for y := region.Y*data.MemoryRegionSize - reach; y < (region.Y+1)*data.MemoryRegionSize+reach; y++ {
			for x := region.X*data.MemoryRegionSize - reach; x < (region.X+1)*data.MemoryRegionSize+reach; x++ {
				tile := utils.Vec2i{X: x, Y: y}
				if _, ok := s.Chunks.RecallFeature(tile); !ok {
					continue
				}
				at := s.Chunks.RememberedFeatureSprites([2]utils.Vec2f{utils.TilePosToWorldPos(tile), utils.TilePosToWorldPos(tile)})
				if len(at) != 1 || !slices.Contains(sprites, at[0]) {
					t.Fatalf("the remembered feature on %v wasn't drawn into region %v, which its sprite may reach", tile, region)
				}
			}
		}
	}

	if !s.ECS.DamageFeature(tree, 1000) {
		t.Fatal("couldn't fell the tree")
	}
	see(s, treePos)

	want := []utils.Vec2i{}
	for dy := -data.MemoryFeatureReachTiles; dy <= data.MemoryFeatureReachTiles; dy++ {
		for dx := -data.MemoryFeatureReachTiles; dx <= data.MemoryFeatureReachTiles; dx++ {
			r := regionAt(utils.TilePosToWorldPos(utils.Vec2i{X: tree.X + dx, Y: tree.Y + dy}))
			if !slices.Contains(want, r) && remembersAnything(s, r) {
				want = append(want, r)
			}
		}
	}
	if got := b.drawn(s, dst, camera); !sameRegions(got, want) {
		t.Fatalf("after remembering the stump, regions %v were baked, want every remembered region its sprite may reach %v", got, want)
	}
}
