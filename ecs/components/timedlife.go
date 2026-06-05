package components

type TimedLife struct {
	remainingMs int64
}

func (TimedLife) isComponent() {}

func (x TimedLife) Copy() TimedLife {
	return TimedLife{
		remainingMs: x.remainingMs,
	}
}
