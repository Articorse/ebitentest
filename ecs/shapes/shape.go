package shapes

import (
	"ebittest/utils"
	"fmt"
	"math/rand/v2"
)

type ShapeEnum uint8

const (
	Shape_Circle ShapeEnum = iota
	Shape_Rectangle
	Shape_Polygon
)

type ShapeParams interface {
	isShapeParams()
}

type Shape interface {
	GetAABB() [2]utils.Vec2f
	GetOffset() utils.Vec2f
	Copy() Shape
	GetRandomPoint(r *rand.Rand) utils.Vec2f
	GetRandomPointAroundShape(r *rand.Rand) utils.Vec2f
}

func CalculateCenter(colShapes []Shape) utils.Vec2f {
	if len(colShapes) == 0 {
		return utils.Vec2f{X: 0, Y: 0}
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

	return utils.Vec2f{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}
}

type ShapeDto struct {
	Type   ShapeEnum
	Params ShapeParams
}

func (x ShapeDto) ToShape() (Shape, error) {
	switch x.Type {
	case Shape_Circle:
		if params, ok := x.Params.(CircleParams); ok {
			r, err := NewCircleShape(params.Radius, params.Offset)
			if err != nil {
				return nil, fmt.Errorf("failed to create circle shape: %w", err)
			}
			return r, nil
		}
	case Shape_Rectangle:
		if params, ok := x.Params.(RectangleParams); ok {
			r, err := NewRectangleShapeFromCoords(params.TopLeft, params.BottomRight, params.Offset)
			if err != nil {
				return nil, fmt.Errorf("failed to create rectangle shape: %w", err)
			}
			return r, nil
		}
	case Shape_Polygon:
		if params, ok := x.Params.(PolygonParams); ok {
			r, err := NewPolygonShape(params.Vertices, params.Offset)
			if err != nil {
				return nil, fmt.Errorf("failed to create polygon shape: %w", err)
			}
			return r, nil
		}
	default:
		return nil, fmt.Errorf("unsupported shape type: %v", x.Type)
	}

	return nil, fmt.Errorf("invalid shape parameters for type: %v", x.Type)
}

func ShapeToDto[T Shape](s T) (ShapeDto, error) {
	switch shape := any(s).(type) {
	case *CircleShape:
		return ShapeDto{
			Type: Shape_Circle,
			Params: CircleParams{
				Radius: shape.radius,
				Offset: shape.offset,
			},
		}, nil
	case *RectangleShape:
		return ShapeDto{
			Type: Shape_Rectangle,
			Params: RectangleParams{
				TopLeft:     shape.topLeft,
				BottomRight: shape.bottomRight,
				Offset:      shape.offset,
			},
		}, nil
	case *PolygonShape:
		return ShapeDto{
			Type: Shape_Polygon,
			Params: PolygonParams{
				Vertices: shape.vertices,
				Offset:   shape.offset,
			},
		}, nil
	default:
		return ShapeDto{}, fmt.Errorf("unsupported shape type for conversion to DTO")
	}
}

func DtoToShape[T ShapeDto](dto ShapeDto) (Shape, error) {
	switch dto.Type {
	case Shape_Circle:
		if params, ok := dto.Params.(*CircleParams); ok {
			return NewCircleShape(params.Radius, params.Offset)
		}
	case Shape_Rectangle:
		if params, ok := dto.Params.(*RectangleParams); ok {
			return NewRectangleShapeFromCoords(params.TopLeft, params.BottomRight, params.Offset)
		}
	case Shape_Polygon:
		if params, ok := dto.Params.(*PolygonParams); ok {
			return NewPolygonShape(params.Vertices, params.Offset)
		}
	default:
		return nil, fmt.Errorf("unsupported shape type: %v", dto.Type)
	}

	return nil, fmt.Errorf("invalid shape parameters for type: %v", dto.Type)
}
