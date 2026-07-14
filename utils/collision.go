package utils

func DetectAABBCollision(a, b [2]Vec2f) bool {
	minAx := a[0].X
	minAy := a[0].Y
	maxAx := a[0].X
	maxAy := a[0].Y
	for _, v := range a {
		if v.X < minAx {
			minAx = v.X
		}
		if v.X > maxAx {
			maxAx = v.X
		}
		if v.Y < minAy {
			minAy = v.Y
		}
		if v.Y > maxAy {
			maxAy = v.Y
		}
	}

	minBx := b[0].X
	minBy := b[0].Y
	maxBx := b[0].X
	maxBy := b[0].Y
	for _, v := range b {
		if v.X < minBx {
			minBx = v.X
		}
		if v.X > maxBx {
			maxBx = v.X
		}
		if v.Y < minBy {
			minBy = v.Y
		}
		if v.Y > maxBy {
			maxBy = v.Y
		}
	}

	return minAx <= maxBx && maxAx >= minBx && minAy <= maxBy && maxAy >= minBy
}

func PointInAABB(p Vec2f, aabb [2]Vec2f) bool {
	minAx := aabb[0].X
	minAy := aabb[0].Y
	maxAx := aabb[0].X
	maxAy := aabb[0].Y
	for _, v := range aabb {
		if v.X < minAx {
			minAx = v.X
		}
		if v.X > maxAx {
			maxAx = v.X
		}
		if v.Y < minAy {
			minAy = v.Y
		}
		if v.Y > maxAy {
			maxAy = v.Y
		}
	}

	return p.X >= minAx && p.X <= maxAx && p.Y >= minAy && p.Y <= maxAy
}
