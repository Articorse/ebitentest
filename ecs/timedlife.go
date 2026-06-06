package ecs

type timedLife struct {
	remainingMs int64
}

func (timedLife) isComponent() {}

func (x timedLife) Copy() timedLife {
	return timedLife{
		remainingMs: x.remainingMs,
	}
}
