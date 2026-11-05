package tilesystem_test

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/ecs/systems/visionsystem"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"testing"
)

// The player and 20 enemies around it, which is as many as spawn
func visionSession(b *testing.B) (*testutil.Session, []common.EntityId) {
	testutil.TempWorld(b)
	s := testutil.StartSession(b)
	s.ECS.AddComponent(s.Player, ecs.NewVisionComponent(data.PlayerVisionRangeTiles, 1))
	viewers := []common.EntityId{s.Player}
	for i := range 20 {
		pos := testutil.PlayerStart.Add(utils.Vec2f{X: float64(i%5-2) * 160, Y: float64(i/5-2) * 160})
		viewers = append(viewers, testutil.AddMovable(b, s.ECS, pos, false, ecs.NewVisionComponent(data.PlayerVisionRangeTiles/2, data.EnemyVisionEveryTicks)))
	}
	return s, viewers
}

// Nobody moves, so nothing has to be seen again
func BenchmarkVisionStill(b *testing.B) {
	s, _ := visionSession(b)
	for b.Loop() {
		visionsystem.Tick(s.ECS)
	}
}

// Everyone steps onto another tile every tick, so everything has to be seen again
func BenchmarkVisionMoving(b *testing.B) {
	s, viewers := visionSession(b)
	step := 1.0
	for b.Loop() {
		step = -step
		for _, e := range viewers {
			testutil.SetPos(b, s.ECS, e, testutil.WorldPos(b, s.ECS, e).Add(utils.Vec2f{X: step * data.TileSize}))
		}
		visionsystem.Tick(s.ECS)
	}
}
