package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
)

// Marks the colony's center, whose chunkloader keeps the colony in the simulation wherever the player is. The chunks
// it loads are the colony's bounds, which is where colonists find their work.
type Colony struct{}

func NewColonyComponent() *Colony {
	return &Colony{}
}

// The first and last tiles of the chunks a loader at pos activates
func ChunkLoaderTiles(pos utils.Vec2f, radius int) (first, last utils.Vec2i) {
	center := utils.WorldPosToChunkGridPos(pos)
	first = utils.Vec2i{X: (center.X - radius) * data.ChunkSize, Y: (center.Y - radius) * data.ChunkSize}
	last = utils.Vec2i{X: (center.X+radius+1)*data.ChunkSize - 1, Y: (center.Y+radius+1)*data.ChunkSize - 1}
	return first, last
}

// The first and last tiles of the colony whose center e is
func (x *ECSContainer) ColonyBounds(e common.EntityId) (first, last utils.Vec2i, err error) {
	loader, ok := x.ChunkLoaders.Get(e)
	if !ok || !x.Colonies.HasComponent(e) {
		return utils.Vec2i{}, utils.Vec2i{}, fmt.Errorf("entity %d isn't a colony center with a chunkloader", e)
	}
	pos, err := x.WorldPos(e)
	if err != nil {
		return utils.Vec2i{}, utils.Vec2i{}, fmt.Errorf("error getting the world position of colony %d: %w", e, err)
	}
	first, last = ChunkLoaderTiles(pos, loader.Radius())
	return first, last, nil
}

func (x *ECSContainer) InColony(e common.EntityId, t utils.Vec2i) bool {
	first, last, err := x.ColonyBounds(e)
	return err == nil && t.X >= first.X && t.X <= last.X && t.Y >= first.Y && t.Y <= last.Y
}

func (x Colony) Copy() Colony {
	return x
}

func (x Colony) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Colonies.addComponent(e, x.Copy())
}

type colonyDto struct{}

func (x Colony) toDto() (ComponentDto, error) {
	return &colonyDto{}, nil
}

func (x colonyDto) toComponent() (Component, error) {
	return &Colony{}, nil
}
