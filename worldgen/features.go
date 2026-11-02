package worldgen

import (
	"ebittest/data"
	"ebittest/utils"
	"ebittest/worldgen/noise"
)

const (
	forestSeed uint64 = iota + 100
	treeChanceSeed
	treePrioritySeed
)

// Like tiles, every feature depends only on the seed and its own position, and the tiles worldgen put around it
func Feature(seed uint64, tilePos utils.Vec2i) data.FeatureEnum {
	if newTreePlacer(seed).isTree(tilePos) {
		return data.Feature_Tree
	}
	return data.Feature_None
}

// The chunk's features, by tile; tiles without one are left out
func ChunkFeatures(seed uint64, chunkPos utils.Vec2i) map[utils.Vec2i]data.FeatureEnum {
	placer := newTreePlacer(seed)
	features := map[utils.Vec2i]data.FeatureEnum{}
	for y := range data.ChunkSize {
		for x := range data.ChunkSize {
			t := utils.Vec2i{X: chunkPos.X*data.ChunkSize + x, Y: chunkPos.Y*data.ChunkSize + y}
			if placer.isTree(t) {
				features[t] = data.Feature_Tree
			}
		}
	}
	return features
}

// Remembers what it worked out, since deciding one tile asks about its neighbours, and theirs
type treePlacer struct {
	seed       uint64
	candidates map[utils.Vec2i]bool
	trees      map[utils.Vec2i]bool
}

func newTreePlacer(seed uint64) *treePlacer {
	return &treePlacer{seed: seed, candidates: map[utils.Vec2i]bool{}, trees: map[utils.Vec2i]bool{}}
}

// A candidate gets a tree unless a neighbour that outranks it got one. Only being blocked by actual trees, rather than
// by every outranking candidate, packs dense forests about twice as tightly. Asking about a neighbour only ever goes to
// one that outranks this tile, so it always ends.
func (p *treePlacer) isTree(t utils.Vec2i) bool {
	if tree, ok := p.trees[t]; ok {
		return tree
	}

	tree := p.isCandidate(t)
	if tree {
		for _, n := range neighbors(t) {
			if p.isCandidate(n) && outranks(p.seed, n, t) && p.isTree(n) {
				tree = false
				break
			}
		}
	}

	p.trees[t] = tree
	return tree
}

func (p *treePlacer) isCandidate(t utils.Vec2i) bool {
	candidate, ok := p.candidates[t]
	if !ok {
		candidate = isTreeCandidate(p.seed, t)
		p.candidates[t] = candidate
	}
	return candidate
}

// Grass where a forest is dense enough that the tile drew a tree. Neighbouring candidates are thinned out afterwards.
func isTreeCandidate(seed uint64, t utils.Vec2i) bool {
	if Tile(seed, t) != data.Tile_Grass {
		return false
	}

	forest := noise.OpenSimplex2(int64(seed+forestSeed), float64(t.X)/data.WorldgenForestScaleTiles, float64(t.Y)/data.WorldgenForestScaleTiles)
	// Forests start where the noise passes the edge, and fill in over the fade, so that they have clear borders
	density := min(max((forest-data.WorldgenForestEdge)/data.WorldgenForestFadeWidth, 0), 1)
	chance := density * data.WorldgenForestMaxTreeChance

	return noise.UnitFloat(noise.Hash2(seed+treeChanceSeed, int64(t.X), int64(t.Y))) < chance
}

// Which of two neighbouring candidates keeps its tree. Ties, which are practically impossible, go by position, so that
// exactly one of them always wins.
func outranks(seed uint64, a, b utils.Vec2i) bool {
	pa := noise.Hash2(seed+treePrioritySeed, int64(a.X), int64(a.Y))
	pb := noise.Hash2(seed+treePrioritySeed, int64(b.X), int64(b.Y))
	if pa != pb {
		return pa > pb
	}
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.X < b.X
}

func neighbors(t utils.Vec2i) [8]utils.Vec2i {
	return [8]utils.Vec2i{
		{X: t.X - 1, Y: t.Y - 1}, {X: t.X, Y: t.Y - 1}, {X: t.X + 1, Y: t.Y - 1},
		{X: t.X - 1, Y: t.Y}, {X: t.X + 1, Y: t.Y},
		{X: t.X - 1, Y: t.Y + 1}, {X: t.X, Y: t.Y + 1}, {X: t.X + 1, Y: t.Y + 1},
	}
}
