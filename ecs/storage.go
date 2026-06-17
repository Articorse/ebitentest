package ecs

import (
	"ebittest/ecs/common"
	"fmt"
	"slices"
)

type Storage[T component] struct {
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
	x.order = slices.DeleteFunc(x.order, func(id common.EntityId) bool {
		return id == e
	})
	delete(x.getData(), e)
}

func (x *Storage[T]) addComponent(e common.EntityId, c T) {
	x.order = append(x.order, e)
	x.data[e] = &c
}
