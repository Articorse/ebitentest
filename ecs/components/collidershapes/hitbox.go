package collidershapes

import "ebittest/utils"

type Shape interface {
	GetAABB() [2]utils.Vec2
	GetOffset() utils.Vec2
	isShape()
}

func CalculateCenter(colShapes []Shape) utils.Vec2 {
	if len(colShapes) == 0 {
		return utils.Vec2{X: 0, Y: 0}
	}

	var minX, minY, maxX, maxY float64
	firstAABB := colShapes[0].GetAABB()
	minX, minY = firstAABB[0].X, firstAABB[0].Y
	maxX, maxY = firstAABB[1].X, firstAABB[1].Y

	for _, shape := range colShapes {
		aabb := shape.GetAABB()
		if aabb[0].X < minX {
			minX = aabb[0].X
		}
		if aabb[0].Y < minY {
			minY = aabb[0].Y
		}
		if aabb[1].X > maxX {
			maxX = aabb[1].X
		}
		if aabb[1].Y > maxY {
			maxY = aabb[1].Y
		}
	}

	return utils.Vec2{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}
}
