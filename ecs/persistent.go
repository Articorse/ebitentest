package ecs

type persistent struct {
	// TODO: Add disable/despawn range
}

func (persistent) isComponent() {}

func (x persistent) Copy() persistent {
	return persistent{}
}

type persistentDto struct {
}

func (persistentDto) isComponentDto() {}

func (x persistent) ToDto() persistentDto {
	return persistentDto{}
}

func (x persistentDto) ToComponent() *persistent {
	return &persistent{}
}
