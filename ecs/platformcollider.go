package ecs

type platformCollider struct {
	baseCollider
}

func (x *platformCollider) getBaseCollider() *baseCollider { return &x.baseCollider }

func (platformCollider) isComponent() {}
