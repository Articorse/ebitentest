package ecs

type hitpoints struct {
	max     int64
	current int64
}

func (*hitpoints) isComponent() {}

func (x hitpoints) Copy() hitpoints {
	return hitpoints{
		max:     x.max,
		current: x.current,
	}
}
