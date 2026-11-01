package tilesystem_test

import (
	"crypto/sha256"
	"ebittest/tests/testutil"
	"ebittest/utils"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Walks the player out and back across many chunks, and fingerprints which entities are simulated in which order
func runScriptedWalk(t *testing.T, worldDir string, ioJitter *rand.Rand) string {
	t.Chdir(worldDir)
	s := testutil.StartSession(t)
	h := sha256.New()

	for i := range 900 {
		step := float64(i % 450)
		if i%450 >= 225 {
			step = 450 - step
		}
		s.MovePlayer(utils.Vec2f{X: 100 + step*6, Y: 100 + step*3})

		if ioJitter != nil && ioJitter.IntN(4) == 0 {
			time.Sleep(time.Duration(ioJitter.IntN(3000)) * time.Microsecond)
		}
		s.Tick(1)

		fmt.Fprintf(h, "%d:%v\n", s.ECS.TickIdx, s.ECS.Transforms.GetEntities())
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestChunkStreamingIsDeterministicUnderIoTiming(t *testing.T) {
	base := testutil.TempWorld(t)
	s := testutil.StartSession(t)
	for i := range 300 {
		e := testutil.AddMovable(t, s.ECS, testutil.PlayerStart, true)
		testutil.SetPos(t, s.ECS, e, utils.Vec2f{X: float64(i * 37 % 1800), Y: float64(i * 53 % 900)})
	}
	s.Tick(2)
	s.Commit()

	runs := map[string]*rand.Rand{
		"plain":      nil,
		"jittered a": rand.New(rand.NewPCG(1, 2)),
		"jittered b": rand.New(rand.NewPCG(3, 4)),
	}
	fingerprints := map[string]string{}
	for name, jitter := range runs {
		dir := filepath.Join(t.TempDir(), "world")
		if err := os.CopyFS(dir, os.DirFS(base)); err != nil {
			t.Fatal(err)
		}
		fingerprints[name] = runScriptedWalk(t, dir, jitter)
	}

	for name, fp := range fingerprints {
		if fp != fingerprints["plain"] {
			t.Errorf("%s run simulated different entities or orders than the plain run", name)
		}
	}
}
