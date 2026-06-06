package ecs

import (
	"ebittest/utils"
)

type spawner struct {
	offset     utils.Vec2
	components []component
}

func (spawner) isComponent() {}

func (x spawner) Copy() spawner {
	componentsCopy := make([]component, len(x.components))
	copy(componentsCopy, x.components)

	return spawner{
		offset:     x.offset,
		components: componentsCopy,
	}
}
