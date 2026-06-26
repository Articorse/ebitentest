package ecs

type chunkLoader struct {
	radius int
}

func (chunkLoader) isComponent() {}

func (x *chunkLoader) Copy() chunkLoader {
	return chunkLoader{
		radius: x.radius,
	}
}
