package ecs

type hitpoints struct {
	max        int64
	current    int64
	invulMaxMs int64
	invulCurMs int64
}

func (hitpoints) isComponent() {}

func (x hitpoints) Copy() hitpoints {
	return hitpoints{
		max:        x.max,
		current:    x.current,
		invulMaxMs: x.invulMaxMs,
		invulCurMs: x.invulCurMs,
	}
}
