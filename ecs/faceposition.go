package ecs

import "ebittest/utils"

type facePosition struct {
	enabled bool
	pos     utils.Vec2
}

func (facePosition) isComponent() {}

func (x facePosition) Copy() facePosition {
	return facePosition{
		enabled: x.enabled,
		pos:     x.pos,
	}
}
