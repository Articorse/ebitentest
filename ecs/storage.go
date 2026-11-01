package ecs

import (
	"ebittest/ecs/common"
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

func newStorage[T Component]() Storage[T] {
	return Storage[T]{order: []common.EntityId{}, data: make(map[common.EntityId]*T)}
}

func (x *Storage[T]) Get(e common.EntityId) (*T, bool) {
	c, ok := x.data[e]
	return c, ok
}

func (x *Storage[T]) deleteEntity(e common.EntityId) {
	if i, found := slices.BinarySearch(x.order, e); found {
		x.order = slices.Delete(x.order, i, i+1)
	}
	delete(x.data, e)
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

type componentStorage interface {
	GetEntities() []common.EntityId
	HasComponent(e common.EntityId) bool
	deleteEntity(e common.EntityId)
	toDto(e common.EntityId) (dto ComponentDto, ok bool, err error)
}

func (x *Storage[T]) toDto(e common.EntityId) (ComponentDto, bool, error) {
	c, ok := x.data[e]
	if !ok {
		return nil, false, nil
	}

	dto, err := (*c).toDto()
	return dto, true, err
}
