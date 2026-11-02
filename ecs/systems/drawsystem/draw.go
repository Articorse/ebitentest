package drawsystem

import (
	"bytes"
	"cmp"
	"ebittest/assetmanager"
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Sprites in view, drawn layer by layer. Within a layer, what touches the ground further down the screen is drawn over
// what's behind it, whether it's an entity or a static sprite such as a tree.
func DrawSprites(
	screen *ebiten.Image,
	camera utils.Vec2f,
	staticSprites []common.StaticSprite,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	order, err := SpriteDrawOrder(camera, staticSprites, ecsContainer, assetManager)
	if err != nil {
		return err
	}

	for _, d := range order {
		if d.Entity == -1 {
			if err := drawStaticSprite(screen, camera, d.Static, assetManager); err != nil {
				return err
			}
			continue
		}

		if err := drawEntitySprite(screen, camera, d.Entity, ecsContainer, assetManager); err != nil {
			return err
		}
	}

	return nil
}

// Where sprites have to be to be drawn. It reaches past the screen, since a sprite's position is its center.
func DrawWindow(camera utils.Vec2f) [2]utils.Vec2f {
	return [2]utils.Vec2f{
		{X: camera.X - data.SpatialHashGridCellSize, Y: camera.Y - data.SpatialHashGridCellSize},
		{X: camera.X + data.CameraWidth + data.SpatialHashGridCellSize, Y: camera.Y + data.CameraHeight + data.SpatialHashGridCellSize},
	}
}

type SpriteDraw struct {
	// -1 for a static sprite
	Entity common.EntityId
	Static common.StaticSprite
}

// An entity together with the entities in the same layer attached to it, drawn one after the other, or a static sprite
type spriteGroup struct {
	key     int64
	root    common.EntityId
	members []common.EntityId
	// The static sprite's place among the static sprites given
	staticIdx int
}

func SpriteDrawOrder(
	camera utils.Vec2f,
	staticSprites []common.StaticSprite,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) ([]SpriteDraw, error) {
	window := DrawWindow(camera)
	inWindow := func(p utils.Vec2f) bool {
		return p.X >= window[0].X && p.X <= window[1].X && p.Y >= window[0].Y && p.Y <= window[1].Y
	}

	visible := map[common.EntityId]uint8{}
	for e, spr := range ecs.Query1(&ecsContainer.Sprites, ecs.Without(&ecsContainer.Disabled)) {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			return nil, fmt.Errorf("error getting world position of entity %d: %v", e, err)
		}
		if !inWindow(pos) {
			continue
		}

		img, err := spr.CurrentFrame(assetManager)
		if err != nil {
			return nil, fmt.Errorf("error getting sprite image for entity %d: %v", e, err)
		}
		if img != nil {
			visible[e] = spr.Layer()
		}
	}

	type member struct {
		e     common.EntityId
		depth int
	}
	groupMembers := map[common.EntityId][]member{}
	for e, layer := range visible {
		root, depth := e, 0
		for p := ecsContainer.Parent(root); p != -1; p = ecsContainer.Parent(root) {
			if pLayer, ok := visible[p]; !ok || pLayer != layer {
				break
			}
			root, depth = p, depth+1
		}
		groupMembers[root] = append(groupMembers[root], member{e, depth})
	}

	layers := map[uint8][]spriteGroup{}
	for root, members := range groupMembers {
		// Parents before their children, so that what's held is drawn over what holds it
		slices.SortFunc(members, func(a, b member) int {
			return cmp.Or(cmp.Compare(a.depth, b.depth), cmp.Compare(b.e, a.e))
		})

		key, err := ecsContainer.SpriteWorldLayerYOffset(root, assetManager)
		if err != nil {
			return nil, fmt.Errorf("error getting the ground position of entity %d's sprite: %v", root, err)
		}

		g := spriteGroup{key: key, root: root}
		for _, m := range members {
			g.members = append(g.members, m.e)
		}
		layers[visible[root]] = append(layers[visible[root]], g)
	}

	for i, s := range staticSprites {
		if !inWindow(s.Pos) {
			continue
		}

		asset, err := assetManager.GetAsset(s.ImageAsset)
		if err != nil {
			return nil, fmt.Errorf("error getting image of static sprite at %v: %v", s.Pos, err)
		}

		key := int64(math.Floor(s.Pos.Y + float64(asset.LayerYOffset)))
		layers[s.Layer] = append(layers[s.Layer], spriteGroup{key: key, root: -1, staticIdx: i})
	}

	order := []SpriteDraw{}
	for _, layer := range slices.Sorted(maps.Keys(layers)) {
		groups := layers[layer]
		// On the same row, static sprites go behind entities, and newer entities behind older ones
		slices.SortFunc(groups, func(a, b spriteGroup) int {
			if c := cmp.Compare(a.key, b.key); c != 0 {
				return c
			}
			if (a.root == -1) != (b.root == -1) {
				if a.root == -1 {
					return -1
				}
				return 1
			}
			if a.root == -1 {
				return cmp.Compare(a.staticIdx, b.staticIdx)
			}
			return cmp.Compare(b.root, a.root)
		})

		for _, g := range groups {
			if g.root == -1 {
				order = append(order, SpriteDraw{Entity: -1, Static: staticSprites[g.staticIdx]})
				continue
			}
			for _, e := range g.members {
				order = append(order, SpriteDraw{Entity: e})
			}
		}
	}

	return order, nil
}

func drawStaticSprite(screen *ebiten.Image, camera utils.Vec2f, s common.StaticSprite, assetManager *assetmanager.AssetManager) error {
	asset, err := assetManager.GetAsset(s.ImageAsset)
	if err != nil {
		return fmt.Errorf("error getting image of static sprite at %v: %v", s.Pos, err)
	}

	img := asset.Frames[0]
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(s.Pos.X-camera.X-float64(img.Bounds().Dx())/2, s.Pos.Y-camera.Y-float64(img.Bounds().Dy())/2)
	screen.DrawImage(img, &opts)

	return nil
}

func drawEntitySprite(
	screen *ebiten.Image,
	camera utils.Vec2f,
	e common.EntityId,
	ecsContainer *ecs.ECSContainer,
	assetManager *assetmanager.AssetManager,
) error {
	spr, ok := ecsContainer.Sprites.Get(e)
	if !ok {
		return fmt.Errorf("entity %d has no sprite", e)
	}

	v, err := ecsContainer.SpriteWorldOffsetPos(e)
	if err != nil {
		return fmt.Errorf("error getting ecs offset position of entity %d: %v", e, err)
	}

	r := 0.0
	if spr.AllowRotation() {
		r, err = ecsContainer.SpriteWorldOffsetRotation(e)
		if err != nil {
			return fmt.Errorf("error getting ecs offset rotation of entity %d: %v", e, err)
		}
	}

	s, err := ecsContainer.SpriteWorldOffsetScale(e)
	if err != nil {
		return fmt.Errorf("error getting ecs offset scale of entity %d: %v", e, err)
	}

	img, err := spr.CurrentFrame(assetManager)
	if err != nil {
		return fmt.Errorf("error getting sprite image for entity %d: %v", e, err)
	}

	opts := ebiten.DrawImageOptions{}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts.GeoM.Scale(s, s)
	opts.GeoM.Translate(-float64(w)*s/2, -float64(h)*s/2)
	if sw, ok := ecsContainer.Swings.Get(e); ok {
		applySwing(&opts.GeoM, sw, ecsContainer.SwingAngle(e), w, h, s)
	}
	opts.GeoM.Rotate(r)
	opts.GeoM.Translate(v.X-camera.X, v.Y-camera.Y)

	if color, ok := spr.CurrentColor(); ok {
		opts.ColorScale.Scale(color.R, color.G, color.B, color.A)
	}

	screen.DrawImage(img, &opts)
	return nil
}

func DrawFloatingTexts(screen *ebiten.Image, ecs *ecs.ECSContainer) error {
	for _, e := range ecs.FloatingTexts.GetEntities() {
		if ecs.IsDisabled(e) {
			continue
		}
		ecsPos, err := ecs.WorldPos(e)
		if err != nil {
			return fmt.Errorf("error getting ecs position of floating text entity %d: %v", e, err)
		}

		ft, _ := ecs.FloatingTexts.Get(e)
		offset, textValue, color, size := ft.Offset(), ft.Text(), ft.Color(), ft.Size()

		face, err := floatingTextFace(size)
		if err != nil {
			return fmt.Errorf("error getting font for floating text entity %d: %v", e, err)
		}

		op := &text.DrawOptions{}
		tX := ecsPos.X + offset.X - ecs.Camera.X
		tY := ecsPos.Y + offset.Y - ecs.Camera.Y
		w, h := text.Measure(textValue, face, 0)
		op.GeoM.Translate(tX-w/2, tY-h/2)
		op.ColorScale.ScaleWithColor(color)

		text.Draw(screen, textValue, face, op)
	}

	return nil
}

const floatingTextFontPath = "/usr/share/fonts/noto/NotoSansMono-Black.ttf"

// Loaded once: parsing the font is slow, and drawn glyphs are only reused within the same source
var floatingTextFont struct {
	loaded bool
	source *text.GoTextFaceSource
	err    error
	faces  map[float64]*text.GoTextFace
}

func floatingTextFace(size float64) (*text.GoTextFace, error) {
	f := &floatingTextFont
	if !f.loaded {
		f.loaded = true
		f.faces = make(map[float64]*text.GoTextFace)

		fontBytes, err := os.ReadFile(floatingTextFontPath)
		if err != nil {
			f.err = fmt.Errorf("error reading font: %w", err)
		} else if f.source, err = text.NewGoTextFaceSource(bytes.NewReader(fontBytes)); err != nil {
			f.err = fmt.Errorf("error parsing font: %w", err)
		}
	}
	if f.err != nil {
		return nil, f.err
	}

	face, ok := f.faces[size]
	if !ok {
		face = &text.GoTextFace{Source: f.source, Size: size}
		f.faces[size] = face
	}
	return face, nil
}

// Turns and flips the item around its grip, so that the grip stays where it's held
func applySwing(geoM *ebiten.GeoM, sw *ecs.Swing, angle float64, w, h int, scale float64) {
	grip := sw.Grip().Subtract(utils.Vec2f{X: float64(w) / 2, Y: float64(h) / 2}).Multiply(scale)

	geoM.Translate(-grip.X, -grip.Y)
	if sw.Mirrored() {
		geoM.Scale(-1, 1)
		angle = -angle
	}
	geoM.Rotate(angle)
	geoM.Translate(grip.X, grip.Y)
}
