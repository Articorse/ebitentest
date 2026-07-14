package ecs

import "ebittest/utils"

type facePosition struct {
	enabled bool
	pos     utils.Vec2f
}

func (facePosition) isComponent() {}

func (x facePosition) Copy() facePosition {
	return facePosition{
		enabled: x.enabled,
		pos:     x.pos,
	}
}

type facePositionDto struct {
	Enabled bool
	Pos     utils.Vec2f
}

func (facePositionDto) isComponentDto() {}

func (x facePosition) ToDto() facePositionDto {
	return facePositionDto{
		Enabled: x.enabled,
		Pos:     x.pos,
	}
}

func (x facePositionDto) ToComponent() *facePosition {
	return &facePosition{
		enabled: x.Enabled,
		pos:     x.Pos,
	}
}
