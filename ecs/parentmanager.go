package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
	"slices"
)

type ParentManager struct{}

func NewParentComponent() *parent {
	return &parent{entity: -1}
}

func (*ParentManager) GetEntity(
	e common.EntityId,
	world *World,
) common.EntityId {
	parComp, err := world.Parents.getComponent(e)
	if err != nil {
		return -1
	}

	return parComp.entity
}

func (*ParentManager) Attach(
	c common.EntityId,
	p common.EntityId,
	world *World,
) error {
	tm := TransformManager{}
	pm := ParentManager{}

	parComp, err := world.Parents.getComponent(c)
	if err == nil {
		if err := pm.Detach(c, world); err != nil {
			return fmt.Errorf("error during detach: %v", err)
		}
	}

	parComp.entity = p

	traComp, _ := world.Transforms.getComponent(c)
	pTraComp, _ := world.Transforms.getComponent(p)

	if traComp == nil || pTraComp == nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, world)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, world)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.entity, world)
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
	e common.EntityId,
	world *World,
) error {
	tm := TransformManager{}

	parComp, err := world.Parents.getComponent(e)
	if err != nil {
		return nil
	}

	if parComp.entity == -1 {
		return nil
	}

	traComp, err := world.Transforms.getComponent(e)
	if err != nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, world)
	if err != nil {
		return fmt.Errorf("error getting world position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, world)
	if err != nil {
		return fmt.Errorf("error getting world rotation of parent entity %d: %v", parComp.entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.entity, world)
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
	e common.EntityId,
	world *World,
) error {
	pm := ParentManager{}
	parents := world.Parents.getData()

	for pE, p := range parents {
		if p.entity == e {
			err := pm.Detach(pE, world)
			if err != nil {
				log.Printf("error detaching entity %d from parent entity %d: %v", pE, e, err)
				continue
			}
		}
	}

	return nil
}

func (*ParentManager) GetChildEntities(
	p common.EntityId,
	world *World,
) ([]common.EntityId, error) {
	children := []common.EntityId{}
	parents := world.Parents.getData()

	for c, parComp := range parents {
		if parComp.entity == p {
			children = append(children, c)
		}
	}

	return children, nil
}

func (*ParentManager) GetOrderedHierarchies(
	entities []common.EntityId,
	world *World,
) ([][][]common.EntityId, error) {
	if len(entities) == 0 {
		return [][][]common.EntityId{}, fmt.Errorf("entities slice empty")
	}

	pm := ParentManager{}

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
			currentParent := pm.GetEntity(root, world)

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
			world,
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
	world *World,
) ([][]common.EntityId, error) {
	pm := ParentManager{}

	if len(hierarchy) <= level {
		hierarchy = append(hierarchy, []common.EntityId{})
	}

	hierarchy[level] = append(hierarchy[level], e)
	checkedEntities[e] = struct{}{}

	children, err := pm.GetChildEntities(e, world)
	if err != nil {
		return [][]common.EntityId{}, fmt.Errorf("error getting child entities of entity %d: %v", e, err)
	}

	for _, c := range children {
		if !slices.Contains(entities, c) {
			continue
		}

		hierarchy, err = getChildHierarchyRecursive(c, level+1, hierarchy, checkedEntities, entities, world)
		if err != nil {
			return [][]common.EntityId{}, fmt.Errorf("error getting child hierarchy of child entity %d: %v", c, err)
		}
	}

	return hierarchy, nil
}
