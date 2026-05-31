package components

import (
	"ebittest/ecs/ecscommon"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
	"slices"
)

type ParentManager struct{}

func NewParentComponent() *Parent {
	return &Parent{entity: -1}
}

func (*ParentManager) GetEntity(
	e ecscommon.EntityId,
	parents map[ecscommon.EntityId]*Parent,
) ecscommon.EntityId {
	parComp, ok := parents[e]
	if !ok {
		return -1
	}

	return parComp.entity
}

func (*ParentManager) Attach(
	c ecscommon.EntityId,
	p ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	tm := TransformManager{}
	pm := ParentManager{}

	parComp, ok := parents[c]
	if ok {
		if err := pm.Detach(c, transforms, parents); err != nil {
			return fmt.Errorf("error during detach: %v", err)
		}
	}

	parComp.entity = p

	traComp, _ := transforms[c]
	pTraComp, _ := transforms[p]

	if traComp == nil || pTraComp == nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parComp.entity, err)
	}

	cos := math.Cos(pTraComp.rotation)
	sin := math.Sin(pTraComp.rotation)

	traComp.pos = utils.Vec2{
		X: (traComp.pos.X*cos - traComp.pos.Y*sin) - pWorldPos.X,
		Y: (traComp.pos.X*sin + traComp.pos.Y*cos) - pWorldPos.Y,
	}
	traComp.scale = traComp.scale / pWorldScale
	traComp.rotation = traComp.rotation - pWorldRot

	return nil
}

// If error handling is changed, check Attach()
func (*ParentManager) Detach(
	e ecscommon.EntityId,
	transforms map[ecscommon.EntityId]*Transform,
	parents map[ecscommon.EntityId]*Parent,
) error {
	tm := TransformManager{}

	parComp, ok := parents[e]
	if !ok {
		return nil
	}

	if parComp.entity == -1 {
		return nil
	}

	traComp, ok := transforms[e]
	if !ok {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.entity, transforms, parents)
	if err != nil {
		return fmt.Errorf("error getting world scale of parent entity %d: %v", parComp.entity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2{
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}

	traComp.scale = pWorldScale
	traComp.rotation = pWorldRot

	parComp.entity = -1

	return nil
}

func (*ParentManager) RemoveParentFromAllEntities(
	e ecscommon.EntityId,
	parents map[ecscommon.EntityId]*Parent,
	transforms map[ecscommon.EntityId]*Transform,
) error {
	pm := ParentManager{}

	for pE, p := range parents {
		if p.entity == e {
			err := pm.Detach(pE, transforms, parents)
			if err != nil {
				log.Printf("error detaching entity %d from parent entity %d: %v", pE, e, err)
				continue
			}
		}
	}

	return nil
}

func (*ParentManager) GetChildEntities(
	p ecscommon.EntityId,
	parents map[ecscommon.EntityId]*Parent,
) ([]ecscommon.EntityId, error) {
	children := []ecscommon.EntityId{}

	for c, parComp := range parents {
		if parComp.entity == p {
			children = append(children, c)
		}
	}

	return children, nil
}

func (*ParentManager) GetOrderedHierarchies(
	entities []ecscommon.EntityId,
	parents map[ecscommon.EntityId]*Parent,
) ([][][]ecscommon.EntityId, error) {
	if len(entities) == 0 {
		return [][][]ecscommon.EntityId{}, fmt.Errorf("Entities slice empty")
	}

	pm := ParentManager{}

	checkedEntities := map[ecscommon.EntityId]struct{}{}
	orderedHierarchies := [][][]ecscommon.EntityId{}

	for _, e := range entities {
		if len(checkedEntities) == len(entities) {
			break
		}

		if _, ok := checkedEntities[e]; ok {
			continue
		}

		root := e

		for {
			currentParent := pm.GetEntity(root, parents)

			if currentParent == -1 || !slices.Contains(entities, currentParent) {
				break
			}

			root = currentParent
		}

		hierarchy := [][]ecscommon.EntityId{}
		hierarchy, err := getChildHierarchyRecursive(
			root,
			0,
			hierarchy,
			checkedEntities,
			parents,
			entities,
		)
		if err != nil {
			return [][][]ecscommon.EntityId{},
				fmt.Errorf("error getting child hierarchy of root entity %d: %v", root, err)
		}

		orderedHierarchies = append(orderedHierarchies, hierarchy)
	}

	return orderedHierarchies, nil
}

func getChildHierarchyRecursive(
	e ecscommon.EntityId,
	level int,
	hierarchy [][]ecscommon.EntityId,
	checkedEntities map[ecscommon.EntityId]struct{},
	parents map[ecscommon.EntityId]*Parent,
	entities []ecscommon.EntityId,
) ([][]ecscommon.EntityId, error) {
	pm := ParentManager{}

	if len(hierarchy) <= level {
		hierarchy = append(hierarchy, []ecscommon.EntityId{})
	}

	hierarchy[level] = append(hierarchy[level], e)
	checkedEntities[e] = struct{}{}

	children, err := pm.GetChildEntities(e, parents)
	if err != nil {
		return [][]ecscommon.EntityId{}, fmt.Errorf("error getting child entities of entity %d: %v", e, err)
	}

	for _, c := range children {
		if !slices.Contains(entities, c) {
			continue
		}

		hierarchy, err = getChildHierarchyRecursive(c, level+1, hierarchy, checkedEntities, parents, entities)
		if err != nil {
			return [][]ecscommon.EntityId{}, fmt.Errorf("error getting child hierarchy of child entity %d: %v", c, err)
		}
	}

	return hierarchy, nil
}
