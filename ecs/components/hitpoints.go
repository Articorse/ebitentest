package components

type Hitpoints struct {
	max     int64
	current int64
}

func (*Hitpoints) isComponent() {}

func (x Hitpoints) Copy() Hitpoints {
	return Hitpoints{
		max:     x.max,
		current: x.current,
	}
}
