package ecs

type hitpoints struct {
	max        int
	current    int
	invulMaxMs int
	invulCurMs int
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
