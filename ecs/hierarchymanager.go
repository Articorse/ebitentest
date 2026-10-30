package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
	"slices"
)

type hierarchyManager struct{}

func NewHierarchyComponent() *hierarchy {
	return &hierarchy{parent: -1}
}

func (*hierarchyManager) GetParent(
	e common.EntityId,
	ecsContainer *ECSContainer,
) common.EntityId {
	hierComp, err := ecsContainer.Hierarchies.getComponent(e)
	if err != nil {
		return -1
	}

	return hierComp.parent
}

func (*hierarchyManager) GetChildren(
	e common.EntityId,
	ecsContainer *ECSContainer,
) []common.EntityId {
	hierComp, err := ecsContainer.Hierarchies.getComponent(e)
	if err != nil {
		return []common.EntityId{}
	}

	return hierComp.children
}

func (*hierarchyManager) Attach(
	c common.EntityId,
	p common.EntityId,
	ecsContainer *ECSContainer,
) error {
	tm := transformManager{}
	hm := hierarchyManager{}

	cHierComp, err := ecsContainer.Hierarchies.getComponent(c)
	if err != nil {
		return fmt.Errorf("could not get hierarchy of child entity %d: %v", c, err)
	}

	pHierComp, err := ecsContainer.Hierarchies.getComponent(p)
	if err != nil {
		return fmt.Errorf("could not get hierarchy of parent entity %d: %v", p, err)
	}

	if err := hm.Detach(c, ecsContainer); err != nil {
		return fmt.Errorf("error during detach: %v", err)
	}

	cHierComp.parent = p
	pHierComp.children = append(pHierComp.children, c)

	traComp, _ := ecsContainer.Transforms.getComponent(c)
	pTraComp, _ := ecsContainer.Transforms.getComponent(p)

	if traComp == nil || pTraComp == nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(cHierComp.parent, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", cHierComp.parent, err)
	}

	pWorldRot, err := tm.GetWorldRotation(cHierComp.parent, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", cHierComp.parent, err)
	}

	pWorldScale, err := tm.GetWorldScale(cHierComp.parent, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", cHierComp.parent, err)
	}

	cos := math.Cos(-pWorldRot)
	sin := math.Sin(-pWorldRot)
	relPos := traComp.pos.Subtract(pWorldPos)

	traComp.pos = utils.Vec2f{
		X: relPos.X*cos - relPos.Y*sin,
		Y: relPos.X*sin + relPos.Y*cos,
	}
	traComp.scale = traComp.scale / pWorldScale
	traComp.rotation = traComp.rotation - pWorldRot

	return nil
}

// If error handling is changed, check Attach()
func (*hierarchyManager) Detach(
	e common.EntityId,
	ecsContainer *ECSContainer,
) error {
	cHierComp, err := ecsContainer.Hierarchies.getComponent(e)
	if err != nil {
		return nil
	}

	if cHierComp.parent == -1 {
		return nil
	}

	traComp, _ := ecsContainer.Transforms.getComponent(e)
	pTraComp, _ := ecsContainer.Transforms.getComponent(cHierComp.parent)

	if traComp != nil && pTraComp != nil {
		if err := detachTransform(e, cHierComp.parent, traComp, ecsContainer); err != nil {
			return err
		}
	}

	if pHierComp, err := ecsContainer.Hierarchies.getComponent(cHierComp.parent); err == nil {
		pHierComp.children = slices.DeleteFunc(pHierComp.children, func(c common.EntityId) bool {
			return c == e
		})
	}

	cHierComp.parent = -1

	return nil
}

func detachTransform(
	e common.EntityId,
	p common.EntityId,
	traComp *transform,
	ecsContainer *ECSContainer,
) error {
	tm := transformManager{}

	pWorldPos, err := tm.GetWorldPos(p, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", p, err)
	}

	pWorldRot, err := tm.GetWorldRotation(p, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", p, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2f{
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}

	eWorldScale, err := tm.GetWorldScale(e, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world scale of entity %d: %v", e, err)
	}

	eWorldRot, err := tm.GetWorldRotation(e, ecsContainer)
	if err != nil {
		return fmt.Errorf("error getting world rotation of entity %d: %v", e, err)
	}

	traComp.scale = eWorldScale
	traComp.rotation = eWorldRot

	return nil
}

func (*hierarchyManager) RemoveParentFromAllEntities(
	e common.EntityId,
	ecsContainer *ECSContainer,
) error {
	hm := hierarchyManager{}
	hierarchies := ecsContainer.Hierarchies.getData()

	for hE, h := range hierarchies {
		if h.parent == e {
			err := hm.Detach(hE, ecsContainer)
			if err != nil {
				log.Printf("error detaching entity %d from parent entity %d: %v", hE, e, err)
				continue
			}
		}
	}

	return nil
}

// Returns entity ids in hierarchies ordered by root and depth level
func (*hierarchyManager) GetOrderedHierarchies(
	entities []common.EntityId,
	ecsContainer *ECSContainer,
) ([][][]common.EntityId, error) {
	if len(entities) == 0 {
		return [][][]common.EntityId{}, fmt.Errorf("entities slice empty")
	}

	hm := hierarchyManager{}

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
			currentParent := hm.GetParent(root, ecsContainer)

			if currentParent == -1 || !slices.Contains(entities, currentParent) {
				break
			}

			root = currentParent
		}

		hierarchy := [][]common.EntityId{}
		hierarchy, err := getChildHierarchyRecursive(
			root,
			0,
			hierarchy,
			checkedEntities,
			entities,
			ecsContainer,
		)
		if err != nil {
			return [][][]common.EntityId{},
				fmt.Errorf("error getting child hierarchy of root entity %d: %v", root, err)
		}

		orderedHierarchies = append(orderedHierarchies, hierarchy)
	}

	return orderedHierarchies, nil
}

func getChildHierarchyRecursive(
	e common.EntityId,
	level int,
	hierarchy [][]common.EntityId,
	checkedEntities map[common.EntityId]struct{},
	entities []common.EntityId,
	ecsContainer *ECSContainer,
) ([][]common.EntityId, error) {
	var err error
	hm := hierarchyManager{}

	if len(hierarchy) <= level {
		hierarchy = append(hierarchy, []common.EntityId{})
	}

	hierarchy[level] = append(hierarchy[level], e)
	checkedEntities[e] = struct{}{}

	children := hm.GetChildren(e, ecsContainer)

	for _, c := range children {
		if !slices.Contains(entities, c) {
			continue
		}

		hierarchy, err = getChildHierarchyRecursive(c, level+1, hierarchy, checkedEntities, entities, ecsContainer)
		if err != nil {
			return [][]common.EntityId{}, fmt.Errorf("error getting child hierarchy of child entity %d: %v", c, err)
		}
	}

	return hierarchy, nil
}

func (*hierarchyManager) GetChildrenDFS(
	e common.EntityId,
	ecsContainer *ECSContainer,
) ([]common.EntityId, error) {
	children := []common.EntityId{}
	children, err := getChildrenRecursiveDFS(
		e,
		children,
		ecsContainer,
	)
	if err != nil {
		return []common.EntityId{},
			fmt.Errorf("error getting recursive children of root entity %d: %v", e, err)
	}

	return children, nil
}

func getChildrenRecursiveDFS(
	e common.EntityId,
	children []common.EntityId,
	ecsContainer *ECSContainer,
) ([]common.EntityId, error) {
	var err error
	hm := hierarchyManager{}

	curChildren := hm.GetChildren(e, ecsContainer)

	for _, c := range curChildren {
		children = append(children, c)
		children, err = getChildrenRecursiveDFS(c, children, ecsContainer)
		if err != nil {
			return []common.EntityId{}, fmt.Errorf("error getting recursive children of child entity %d: %v", c, err)
		}
	}

	return children, nil
}
