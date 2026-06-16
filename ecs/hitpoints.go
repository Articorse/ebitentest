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
