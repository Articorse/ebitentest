package components

type Platform struct{}

func (Platform) isComponent() {}

func (x Platform) Copy() Platform {
	return Platform{}
}
