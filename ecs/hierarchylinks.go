package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"math"
	"slices"
)

func (x *ECSContainer) Parent(e common.EntityId) common.EntityId {
	h, ok := x.Hierarchies.Get(e)
	if !ok {
		return -1
	}

	return h.parent
}

func (x *ECSContainer) Children(e common.EntityId) []common.EntityId {
	h, ok := x.Hierarchies.Get(e)
	if !ok {
		return []common.EntityId{}
	}

	return h.Children()
}

func (x *ECSContainer) LinkKind(e common.EntityId) LinkKindEnum {
	h, ok := x.Hierarchies.Get(e)
	if !ok {
		return Link_Part
	}

	return h.linkKind
}

func (x *ECSContainer) Root(e common.EntityId) common.EntityId {
	for p := x.Parent(e); p != -1; p = x.Parent(e) {
		e = p
	}

	return e
}

// The root of the composition e belongs to: the nearest ancestor reached without crossing an attached link
func (x *ECSContainer) CompositionRoot(e common.EntityId) common.EntityId {
	for {
		p := x.Parent(e)
		if p == -1 || x.LinkKind(e) == Link_Attached {
			return e
		}
		e = p
	}
}

func (x *ECSContainer) Attach(c common.EntityId, p common.EntityId, linkKind LinkKindEnum) error {
	cHier, ok := x.Hierarchies.Get(c)
	if !ok {
		return fmt.Errorf("child entity %d has no hierarchy", c)
	}

	pHier, ok := x.Hierarchies.Get(p)
	if !ok {
		return fmt.Errorf("parent entity %d has no hierarchy", p)
	}

	for a := p; a != -1; a = x.Parent(a) {
		if a == c {
			return fmt.Errorf("cannot attach entity %d to entity %d, as it would create a cycle", c, p)
		}
	}

	if err := x.Detach(c); err != nil {
		return fmt.Errorf("error during detach: %v", err)
	}

	cHier.parent = p
	cHier.linkKind = linkKind
	pHier.children = append(pHier.children, c)

	t, ok := x.Transforms.Get(c)
	if !ok || !x.Transforms.HasComponent(p) {
		return nil
	}

	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	pWorldScale, err := x.WorldScale(p)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", p, err)
	}

	cos := math.Cos(-pWorldRot)
	sin := math.Sin(-pWorldRot)
	relPos := t.pos.Subtract(pWorldPos)

	t.pos = utils.Vec2f{
		X: relPos.X*cos - relPos.Y*sin,
		Y: relPos.X*sin + relPos.Y*cos,
	}
	t.scale = t.scale / pWorldScale
	t.rotation = t.rotation - pWorldRot

	return nil
}

// If error handling is changed, check Attach()
func (x *ECSContainer) Detach(e common.EntityId) error {
	h, ok := x.Hierarchies.Get(e)
	if !ok || h.parent == -1 {
		return nil
	}

	t, ok := x.Transforms.Get(e)
	if ok && x.Transforms.HasComponent(h.parent) {
		if err := x.detachTransform(e, h.parent, t); err != nil {
			return err
		}
	}

	x.removeChild(h.parent, e)
	h.parent = -1

	return nil
}

func (x *ECSContainer) removeChild(p common.EntityId, c common.EntityId) {
	pHier, ok := x.Hierarchies.Get(p)
	if !ok {
		return
	}

	pHier.children = slices.DeleteFunc(pHier.children, func(child common.EntityId) bool {
		return child == c
	})
}

func (x *ECSContainer) detachTransform(e common.EntityId, p common.EntityId, t *Transform) error {
	pWorldPos, err := x.WorldPos(p)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := x.WorldRotation(p)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	t.pos = utils.Vec2f{
		X: pWorldPos.X + (t.pos.X*cos - t.pos.Y*sin),
		Y: pWorldPos.Y + (t.pos.X*sin + t.pos.Y*cos),
	}

	eWorldScale, err := x.WorldScale(e)
	if err != nil {
		return fmt.Errorf("error getting world scale of entity %d: %v", e, err)
	}

	eWorldRot, err := x.WorldRotation(e)
	if err != nil {
		return fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	t.scale = eWorldScale
	t.rotation = eWorldRot

	return nil
}

// Returns entity ids in hierarchies ordered by root and depth level
func (x *ECSContainer) OrderedHierarchies(entities []common.EntityId) ([][][]common.EntityId, error) {
	if len(entities) == 0 {
		return [][][]common.EntityId{}, fmt.Errorf("entities slice empty")
	}

	checkedEntities := map[common.EntityId]struct{}{}
	orderedHierarchies := [][][]common.EntityId{}

	for _, e := range entities {
		if len(checkedEntities) == len(entities) {
			break
		}

		if _, ok := checkedEntities[e]; ok {
			continue
		}

		root := e

		for {
			currentParent := x.Parent(root)

			if currentParent == -1 || !slices.Contains(entities, currentParent) {
				break
			}

			root = currentParent
		}

		hierarchy := x.childHierarchy(root, 0, [][]common.EntityId{}, checkedEntities, entities)
		orderedHierarchies = append(orderedHierarchies, hierarchy)
	}

	return orderedHierarchies, nil
}

func (x *ECSContainer) childHierarchy(
	e common.EntityId,
	level int,
	hierarchy [][]common.EntityId,
	checkedEntities map[common.EntityId]struct{},
	entities []common.EntityId,
) [][]common.EntityId {
	if len(hierarchy) <= level {
		hierarchy = append(hierarchy, []common.EntityId{})
	}

	hierarchy[level] = append(hierarchy[level], e)
	checkedEntities[e] = struct{}{}

	for _, c := range x.Children(e) {
		if !slices.Contains(entities, c) {
			continue
		}

		hierarchy = x.childHierarchy(c, level+1, hierarchy, checkedEntities, entities)
	}

	return hierarchy
}

func (x *ECSContainer) ChildrenDFS(e common.EntityId) []common.EntityId {
	children := []common.EntityId{}

	for _, c := range x.Children(e) {
		children = append(children, c)
		children = append(children, x.ChildrenDFS(c)...)
	}

	return children
}
