package components

import (
	"ebittest/utils"
)

type Spawner struct {
	offset     utils.Vec2
	components []Component
}

func (Spawner) isComponent() {}

func (x Spawner) Copy() Spawner {
	componentsCopy := make([]Component, len(x.components))
	copy(componentsCopy, x.components)

	return Spawner{
		offset:     x.offset,
		components: componentsCopy,
	}
}
