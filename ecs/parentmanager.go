package ecs

import (
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"log"
	"math"
	"slices"
)

type parentManager struct{}

func NewParentComponent() *parent {
	return &parent{entity: -1}
}

func (*parentManager) GetEntity(
	e common.EntityId,
	ecs *ECS,
) common.EntityId {
	parComp, err := ecs.Parents.getComponent(e)
	if err != nil {
		return -1
	}

	return parComp.entity
}

func (*parentManager) Attach(
	c common.EntityId,
	p common.EntityId,
	ecs *ECS,
) error {
	tm := transformManager{}
	pm := parentManager{}

	parComp, err := ecs.Parents.getComponent(c)
	if err == nil {
		if err := pm.Detach(c, ecs); err != nil {
			return fmt.Errorf("error during detach: %v", err)
		}
	}

	parComp.entity = p

	traComp, _ := ecs.Transforms.getComponent(c)
	pTraComp, _ := ecs.Transforms.getComponent(p)

	if traComp == nil || pTraComp == nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs rotation of parent entity %d: %v", parComp.entity, err)
	}

	pWorldScale, err := tm.GetWorldScale(parComp.entity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs scale of parent entity %d: %v", parComp.entity, err)
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
func (*parentManager) Detach(
	e common.EntityId,
	ecs *ECS,
) error {
	tm := transformManager{}

	parComp, err := ecs.Parents.getComponent(e)
	if err != nil {
		return nil
	}

	if parComp.entity == -1 {
		return nil
	}

	traComp, err := ecs.Transforms.getComponent(e)
	if err != nil {
		return nil
	}

	pWorldPos, err := tm.GetWorldPos(parComp.entity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs position of parent entity %d: %v", parComp.entity, err)
	}

	pWorldRot, err := tm.GetWorldRotation(parComp.entity, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs rotation of parent entity %d: %v", parComp.entity, err)
	}

	cos := math.Cos(pWorldRot)
	sin := math.Sin(pWorldRot)

	traComp.pos = utils.Vec2{
		X: pWorldPos.X + (traComp.pos.X*cos - traComp.pos.Y*sin),
		Y: pWorldPos.Y + (traComp.pos.X*sin + traComp.pos.Y*cos),
	}

	eWorldScale, err := tm.GetWorldScale(e, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs scale of entity %d: %v", e, err)
	}

	eWorldRot, err := tm.GetWorldRotation(e, ecs)
	if err != nil {
		return fmt.Errorf("error getting ecs rotation of entity %d: %v", e, err)
	}

	traComp.scale = eWorldScale
	traComp.rotation = eWorldRot

	parComp.entity = -1

	return nil
}

func (*parentManager) RemoveParentFromAllEntities(
	e common.EntityId,
	ecs *ECS,
) error {
	pm := parentManager{}
	parents := ecs.Parents.getData()

	for pE, p := range parents {
		if p.entity == e {
			err := pm.Detach(pE, ecs)
			if err != nil {
				log.Printf("error detaching entity %d from parent entity %d: %v", pE, e, err)
				continue
			}
		}
	}

	return nil
}

func (*parentManager) GetChildEntities(
	p common.EntityId,
	ecs *ECS,
) ([]common.EntityId, error) {
	children := []common.EntityId{}
	parents := ecs.Parents.getData()

	for c, parComp := range parents {
		if parComp.entity == p {
			children = append(children, c)
		}
	}

	return children, nil
}

func (*parentManager) GetOrderedHierarchies(
	entities []common.EntityId,
	ecs *ECS,
) ([][][]common.EntityId, error) {
	if len(entities) == 0 {
		return [][][]common.EntityId{}, fmt.Errorf("entities slice empty")
	}

	pm := parentManager{}

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
			currentParent := pm.GetEntity(root, ecs)

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
			ecs,
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
	ecs *ECS,
) ([][]common.EntityId, error) {
	pm := parentManager{}

	if len(hierarchy) <= level {
		hierarchy = append(hierarchy, []common.EntityId{})
	}

	hierarchy[level] = append(hierarchy[level], e)
	checkedEntities[e] = struct{}{}

	children, err := pm.GetChildEntities(e, ecs)
	if err != nil {
		return [][]common.EntityId{}, fmt.Errorf("error getting child entities of entity %d: %v", e, err)
	}

	for _, c := range children {
		if !slices.Contains(entities, c) {
			continue
		}

		hierarchy, err = getChildHierarchyRecursive(c, level+1, hierarchy, checkedEntities, entities, ecs)
		if err != nil {
			return [][]common.EntityId{}, fmt.Errorf("error getting child hierarchy of child entity %d: %v", c, err)
		}
	}

	return hierarchy, nil
}
