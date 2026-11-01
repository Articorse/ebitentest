package ecs

import (
	"ebittest/ecs/common"
	"fmt"
	"slices"
)

// Entities are kept sorted by ID, so that systems iterate in the same order no matter when or how entities were
// added, which a restored world has to match to replay deterministically
type Storage[T Component] struct {
	order []common.EntityId
	data  map[common.EntityId]*T
}

func (x *Storage[T]) GetEntities() []common.EntityId {
	return x.order
}

func (x *Storage[T]) HasComponent(e common.EntityId) bool {
	_, ok := x.data[e]
	return ok
}

func (x *Storage[T]) tryGetComponent(e common.EntityId) (*T, bool) {
	c, ok := x.data[e]
	return c, ok
}

func (x *Storage[T]) getData() map[common.EntityId]*T {
	return x.data
}

func (x *Storage[T]) getComponent(e common.EntityId) (*T, error) {
	c, ok := x.data[e]
	if !ok {
		return nil, fmt.Errorf("could not get %T component of entity %d", *new(T), e)
	}

	return c, nil
}

func (x *Storage[T]) deleteEntity(e common.EntityId) {
	if i, found := slices.BinarySearch(x.order, e); found {
		x.order = slices.Delete(x.order, i, i+1)
	}
	delete(x.getData(), e)
}

func (x *Storage[T]) addComponent(e common.EntityId, c T) {
	if _, ok := x.data[e]; !ok {
		x.insertOrdered(e)
	}
	x.data[e] = &c
}

// New entities have the highest ID and are appended. Anything else is inserted into a copy, since a system may be
// iterating the current slice while it adds a component to an existing entity.
func (x *Storage[T]) insertOrdered(e common.EntityId) {
	i, _ := slices.BinarySearch(x.order, e)
	if i == len(x.order) {
		x.order = append(x.order, e)
		return
	}

	x.order = slices.Concat(x.order[:i], []common.EntityId{e}, x.order[i:])
}
