package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/utils"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Implemented by the params of input sources that walk paths
type PathFollower interface {
	// From the tile the entity is on, or empty while it isn't following one
	CurrentPath() []utils.Vec2i
}

// From each entity following a path, through the centers of the path's tiles after the one it's on
func DrawPaths(world *ebiten.Image, camera utils.Vec2f, ecsContainer *ecs.ECSContainer) {
	for e, in := range ecs.Query1(&ecsContainer.Inputs, ecs.Without(&ecsContainer.Disabled)) {
		follower, ok := in.Params().(PathFollower)
		if !ok {
			continue
		}
		path := follower.CurrentPath()
		if len(path) < 2 {
			continue
		}
		from, err := ecsContainer.WorldPos(e)
		if err != nil {
			continue
		}

		for _, t := range path[1:] {
			to := utils.TilePosToWorldPos(t)
			vector.StrokeLine(world, float32(from.X-camera.X), float32(from.Y-camera.Y), float32(to.X-camera.X), float32(to.Y-camera.Y), 1, data.Debug_PathColor, false)
			from = to
		}
	}
}
