package ecs

import "ebittest/ecs/common"

type hitpoints struct {
	max            int
	current        int
	postHitInvulMs int
	invulCurMs     int
}

func (x hitpoints) Copy() hitpoints {
	return hitpoints{
		max:            x.max,
		current:        x.current,
		postHitInvulMs: x.postHitInvulMs,
		invulCurMs:     x.invulCurMs,
	}
}

func (x hitpoints) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Hitpoints.addComponent(e, x.Copy())
}

type hitpointsDto struct {
	Max            int
	Current        int
	PostHitInvulMs int
	InvulCurMs     int
}

func (x hitpoints) toDto() (ComponentDto, error) {
	return &hitpointsDto{
		Max:            x.max,
		Current:        x.current,
		PostHitInvulMs: x.postHitInvulMs,
		InvulCurMs:     x.invulCurMs,
	}, nil
}

func (x hitpointsDto) toComponent() (Component, error) {
	return &hitpoints{
		max:            x.Max,
		current:        x.Current,
		postHitInvulMs: x.PostHitInvulMs,
		invulCurMs:     x.InvulCurMs,
	}, nil
}
