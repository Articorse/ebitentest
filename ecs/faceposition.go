package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
)

type facePosition struct {
	enabled bool
	pos     utils.Vec2f
}

func (x facePosition) Copy() facePosition {
	return facePosition{
		enabled: x.enabled,
		pos:     x.pos,
	}
}

func (x facePosition) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.FacePositions.addComponent(e, x.Copy())
}

type facePositionDto struct {
	Enabled bool
	Pos     utils.Vec2f
}

func (x facePosition) toDto() (ComponentDto, error) {
	return &facePositionDto{
		Enabled: x.enabled,
		Pos:     x.pos,
	}, nil
}

func (x facePositionDto) toComponent() (Component, error) {
	return &facePosition{
		enabled: x.Enabled,
		pos:     x.Pos,
	}, nil
}
