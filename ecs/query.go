package ecs

import (
	"ebittest/ecs/common"
	"iter"
)

// Narrows a query by whether entities have components, without handing the components out
type Filter struct {
	storages []componentStorage
	present  bool
}

func With(storages ...componentStorage) Filter {
	return Filter{storages: storages, present: true}
}

func Without(storages ...componentStorage) Filter {
	return Filter{storages: storages, present: false}
}

type Row2[A, B Component] struct {
	A *A
	B *B
}

type Row3[A, B, C Component] struct {
	A *A
	B *B
	C *C
}

type Row4[A, B, C, D Component] struct {
	A *A
	B *B
	C *C
	D *D
}

func Query1[A Component](a *Storage[A], filters ...Filter) iter.Seq2[common.EntityId, *A] {
	return func(yield func(common.EntityId, *A) bool) {
		for e := range matching(filters, a) {
			if !yield(e, a.data[e]) {
				return
			}
		}
	}
}

func Query2[A, B Component](a *Storage[A], b *Storage[B], filters ...Filter) iter.Seq2[common.EntityId, Row2[A, B]] {
	return func(yield func(common.EntityId, Row2[A, B]) bool) {
		for e := range matching(filters, a, b) {
			if !yield(e, Row2[A, B]{A: a.data[e], B: b.data[e]}) {
				return
			}
		}
	}
}

func Query3[A, B, C Component](
	a *Storage[A],
	b *Storage[B],
	c *Storage[C],
	filters ...Filter,
) iter.Seq2[common.EntityId, Row3[A, B, C]] {
	return func(yield func(common.EntityId, Row3[A, B, C]) bool) {
		for e := range matching(filters, a, b, c) {
			if !yield(e, Row3[A, B, C]{A: a.data[e], B: b.data[e], C: c.data[e]}) {
				return
			}
		}
	}
}

func Query4[A, B, C, D Component](
	a *Storage[A],
	b *Storage[B],
	c *Storage[C],
	d *Storage[D],
	filters ...Filter,
) iter.Seq2[common.EntityId, Row4[A, B, C, D]] {
	return func(yield func(common.EntityId, Row4[A, B, C, D]) bool) {
		for e := range matching(filters, a, b, c, d) {
			if !yield(e, Row4[A, B, C, D]{A: a.data[e], B: b.data[e], C: c.data[e], D: d.data[e]}) {
				return
			}
		}
	}
}

// Walks the smallest of the required storages, which is in entity-ID order like every other, so the order doesn't
// depend on which storage that is. Each entity is checked when it's reached, so changes made while iterating apply.
func matching(filters []Filter, required ...componentStorage) iter.Seq[common.EntityId] {
	return func(yield func(common.EntityId) bool) {
		driver := required[0]
		for _, s := range required[1:] {
			if len(s.GetEntities()) < len(driver.GetEntities()) {
				driver = s
			}
		}
		for _, f := range filters {
			if !f.present {
				continue
			}
			for _, s := range f.storages {
				if len(s.GetEntities()) < len(driver.GetEntities()) {
					driver = s
				}
			}
		}

		for _, e := range driver.GetEntities() {
			if !hasAll(e, required, filters) {
				continue
			}
			if !yield(e) {
				return
			}
		}
	}
}

func hasAll(e common.EntityId, required []componentStorage, filters []Filter) bool {
	for _, s := range required {
		if !s.HasComponent(e) {
			return false
		}
	}

	for _, f := range filters {
		for _, s := range f.storages {
			if s.HasComponent(e) != f.present {
				return false
			}
		}
	}

	return true
}
