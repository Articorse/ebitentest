package ecs

import (
	"ebittest/data"
	"ebittest/utils"
)

// The light on a square of tiles, worked out for all of them at once, since light spreads from tile to tile
type LightMap struct {
	center utils.Vec2i
	radius int
	levels []uint8

	// Kept between uses, so that working out light every tick doesn't allocate
	steps  []int8
	opaque []bool
	queue  []int
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
// Light reaching a tile can only have come from within data.MaxLight / data.LightFalloffPerTile steps of it, so the
// tiles that far beyond the square are worked out too, and the square's own light is exact.
func (x *ECSContainer) LightAround(center utils.Vec2i, radius int, m *LightMap) {
	reach := data.MaxLight / data.LightFalloffPerTile
	m.center, m.radius = center, radius+reach
	side := 2*m.radius + 1

	m.levels = resize(m.levels, side*side)
	m.steps = resize(m.steps, side*side)
	m.opaque = resize(m.opaque, side*side)
	m.queue = m.queue[:0]

	sky := SkyLight(Darkness(x.TickIdx))
	for row := range side {
		for col := range side {
			i := row*side + col
			t := utils.Vec2i{X: center.X - m.radius + col, Y: center.Y - m.radius + row}
			m.opaque[i] = x.terrain.IsOpaqueTile(t)
			m.steps[i] = -1
			if x.terrain.IsOpenToSky(t) {
				m.steps[i] = 0
				if !m.opaque[i] {
					m.queue = append(m.queue, i)
				}
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
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}
