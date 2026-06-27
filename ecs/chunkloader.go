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

type chunkLoaderDto struct {
	Radius int
}

func (chunkLoaderDto) isComponentDto() {}

func (x chunkLoader) ToDto() chunkLoaderDto {
	return chunkLoaderDto{
		Radius: x.radius,
	}
}

func (x chunkLoaderDto) ToComponent() *chunkLoader {
	return &chunkLoader{
		radius: x.Radius,
	}
}
