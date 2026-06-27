package ecs

type hitpoints struct {
	max            int
	current        int
	postHitInvulMs int
	invulCurMs     int
}

func (hitpoints) isComponent() {}

func (x hitpoints) Copy() hitpoints {
	return hitpoints{
		max:            x.max,
		current:        x.current,
		postHitInvulMs: x.postHitInvulMs,
		invulCurMs:     x.invulCurMs,
	}
}

type hitpointsDto struct {
	Max            int
	Current        int
	PostHitInvulMs int
	InvulCurMs     int
}

func (hitpointsDto) isComponentDto() {}

func (x hitpoints) ToDto() hitpointsDto {
	return hitpointsDto{
		Max:            x.max,
		Current:        x.current,
		PostHitInvulMs: x.postHitInvulMs,
		InvulCurMs:     x.invulCurMs,
	}
}

func (x hitpointsDto) ToComponent() *hitpoints {
	return &hitpoints{
		max:            x.Max,
		current:        x.Current,
		postHitInvulMs: x.PostHitInvulMs,
		invulCurMs:     x.InvulCurMs,
	}
}
