package ecs

import (
	"ebittest/data"
	"ebittest/fov"
	"ebittest/utils"
	"slices"
)

// The light on a square of tiles, worked out for all of them at once, since light spreads from tile to tile
type LightMap struct {
	center utils.Vec2i
	radius int
	levels []uint8

	// Kept between uses, so that working out light every tick doesn't allocate
	steps  []int8
	opaque []bool
	sky    []bool
	queue  []int

	// What the light was worked out from, so that it's only worked out again once something it depends on changes
	key      lightKey
	emitters []heldLight
	// Where the lights carried around are gathered to be compared with the last ones, kept to avoid allocating
	nextEmitters []heldLight
}

type lightKey struct {
	valid        bool
	center       utils.Vec2i
	radius       int
	sky          uint8
	sightVersion uint64
}

type heldLight struct {
	source     utils.Vec2i
	level      uint8
	rangeTiles int
}

// Opaque tiles from the map where it has them, and from the terrain beyond, the same either way
func (m *LightMap) IsOpaque(t utils.Vec2i, terrain Terrain) bool {
	if i, ok := m.index(t.X-m.center.X+m.radius, t.Y-m.center.Y+m.radius); ok && i < len(m.opaque) {
		return m.opaque[i]
	}
	return terrain.IsOpaqueTile(t)
}

// The square the light was worked out for, which reaches past the one asked for
func (m *LightMap) Area() (center utils.Vec2i, radius int) {
	return m.center, m.radius
}

// 0 outside the square, and before the light is first worked out
func (m *LightMap) At(t utils.Vec2i) uint8 {
	i, ok := m.index(t.X-m.center.X+m.radius, t.Y-m.center.Y+m.radius)
	if !ok || i >= len(m.levels) {
		return 0
	}
	return m.levels[i]
}

func (m *LightMap) index(x, y int) (int, bool) {
	side := 2*m.radius + 1
	if x < 0 || y < 0 || x >= side || y >= side {
		return 0, false
	}
	return y*side + x, true
}

// The sky lights the tiles open to it, and its light spreads from them under roofs and into caves, in all eight
// directions, losing data.LightFalloffPerTile with each step. Opaque tiles are lit, so that walls facing the light can be
// seen, but don't pass it on. Where light arrives from several places, the brightest wins.
//
// Features such as torches, and light emitters such as torches in hand, light what they can see, fading with distance
// rather than spreading around corners, so their shadows are sharp.
//
// Sky light reaching a tile can only have come from within data.MaxLight / data.LightFalloffPerTile steps of it, so the
// tiles that far beyond the square are worked out too, and the square's own light is exact.
//
// Nothing is worked out again unless the square, the sky's light, the terrain or the lights carried around it have
// changed since the last time, which is what changed reports.
func (x *ECSContainer) LightAround(center utils.Vec2i, radius int, m *LightMap) (changed bool) {
	reach := data.MaxLight / data.LightFalloffPerTile
	sky := SkyLight(Darkness(x.TickIdx))
	key := lightKey{valid: true, center: center, radius: radius + reach, sky: sky}
	sightTerrain, cacheable := x.terrain.(SightTerrain)
	if cacheable {
		key.sightVersion = sightTerrain.SightVersion()
	}
	emitters := x.heldLightsAround(center, key.radius, m.nextEmitters[:0])
	if cacheable && key == m.key && slices.Equal(emitters, m.emitters) {
		m.nextEmitters = emitters
		return false
	}
	m.key, m.emitters, m.nextEmitters = key, emitters, m.emitters

	m.center, m.radius = center, radius+reach
	side := 2*m.radius + 1

	m.levels = resize(m.levels, side*side)
	m.steps = resize(m.steps, side*side)
	m.opaque = resize(m.opaque, side*side)
	m.sky = resize(m.sky, side*side)
	m.queue = m.queue[:0]

	from := utils.Vec2i{X: center.X - m.radius, Y: center.Y - m.radius}
	if cacheable {
		sightTerrain.TileFlags(from, side, side, m.opaque, m.sky)
	} else {
		for i := range side * side {
			t := utils.Vec2i{X: from.X + i%side, Y: from.Y + i/side}
			m.opaque[i], m.sky[i] = x.terrain.IsOpaqueTile(t), x.terrain.IsOpenToSky(t)
		}
	}
	for i := range side * side {
		m.steps[i] = -1
		if m.sky[i] {
			m.steps[i] = 0
			if !m.opaque[i] {
				m.queue = append(m.queue, i)
			}
		}
	}

	for head := 0; head < len(m.queue); head++ {
		i := m.queue[head]
		if int(m.steps[i]) >= reach {
			continue
		}
		cx, cy := i%side, i/side
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				j, ok := m.index(cx+dx, cy+dy)
				if !ok || m.steps[j] >= 0 {
					continue
				}
				m.steps[j] = m.steps[i] + 1
				if !m.opaque[j] {
					m.queue = append(m.queue, j)
				}
			}
		}
	}

	for i, s := range m.steps {
		m.levels[i] = 0
		if s >= 0 {
			m.levels[i] = uint8(max(int(sky)-int(s)*data.LightFalloffPerTile, 0))
		}
	}

	areaMin := utils.Vec2i{X: center.X - m.radius, Y: center.Y - m.radius}
	areaMax := utils.Vec2i{X: center.X + m.radius, Y: center.Y + m.radius}
	isOpaque := func(t utils.Vec2i) bool { return m.IsOpaque(t, x.terrain) }
	shine := func(source utils.Vec2i, level uint8, rangeTiles int) {
		fov.Compute(source, rangeTiles, isOpaque, func(t utils.Vec2i) {
			i, ok := m.index(t.X-areaMin.X, t.Y-areaMin.Y)
			if !ok {
				return
			}
			if lit := fadingLightAt(t.X-source.X, t.Y-source.Y, level, rangeTiles); lit > m.levels[i] {
				m.levels[i] = lit
			}
		})
	}
	x.terrain.LightSources(areaMin, areaMax, shine)
	for _, l := range m.emitters {
		shine(l.source, l.level, l.rangeTiles)
	}
	return true
}

// The lights carried by someone that reach the square, in entity order, from the tiles of whoever carries them
func (x *ECSContainer) heldLightsAround(center utils.Vec2i, radius int, into []heldLight) []heldLight {
	for e, emitter := range Query1(&x.LightEmitters, Without(&x.Disabled)) {
		if x.LinkKind(e) != Link_Held {
			continue
		}
		pos, err := x.WorldPos(x.Parent(e))
		if err != nil {
			continue
		}

		source, r := utils.WorldPosToTilePos(pos), emitter.rangeTiles
		if utils.AbsInt(source.X-center.X) <= radius+r && utils.AbsInt(source.Y-center.Y) <= radius+r {
			into = append(into, heldLight{source: source, level: emitter.level, rangeTiles: r})
		}
	}
	return into
}

// Fading linearly with distance, to nothing at the light's range. In integers, so that it's the same on every machine:
// level·(range−d)/range rounded down is (level·range − ⌈level·d⌉)/range rounded down, and level·d = √(level²·d²).
func fadingLightAt(dx, dy int, level uint8, rangeTiles int) uint8 {
	l := int(level)
	return uint8(max((l*rangeTiles-utils.CeilSqrt(l*l*(dx*dx+dy*dy)))/rangeTiles, 0))
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}
