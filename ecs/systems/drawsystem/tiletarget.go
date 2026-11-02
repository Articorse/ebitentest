package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	tileTargetColor        = color.White
	tileTargetInvalidColor = color.Black
)

// Outlines the tile the holder's tile-targeting item would hit, or the one under the aim if it wouldn't hit any
func DrawTileTarget(
	screen *ebiten.Image,
	camera utils.Vec2f,
	holder common.EntityId,
	aimPos utils.Vec2f,
	ecsContainer *ecs.ECSContainer,
) error {
	if ecsContainer.IsDisabled(holder) {
		return nil
	}

	targeting, ok := ecsContainer.HeldTileTargeting(holder)
	if !ok {
		return nil
	}

	holderPos, err := ecsContainer.WorldPos(holder)
	if err != nil {
		return err
	}

	tile, ok := targeting.TargetTile(holderPos, aimPos, ecsContainer.Terrain())
	clr := tileTargetColor
	if !ok {
		tile, clr = utils.WorldPosToTilePos(aimPos), tileTargetInvalidColor
	}

	// Strokes are centered on the rectangle's edges, so it's inset by half the width to draw the outline inside the tile
	const width = 2
	topLeft := utils.TilePosToWorldPos(tile).Subtract(utils.Vec2f{X: data.TileSize / 2, Y: data.TileSize / 2}).Subtract(camera)
	vector.StrokeRect(
		screen,
		float32(topLeft.X)+width/2,
		float32(topLeft.Y)+width/2,
		data.TileSize-width,
		data.TileSize-width,
		width,
		clr,
		false,
	)

	return nil
}
