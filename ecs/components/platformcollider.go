package components

type PlatformCollider struct {
	BaseColliderComponent
}

func (x *PlatformCollider) getBaseCollider() *BaseColliderComponent { return &x.BaseColliderComponent }

func (PlatformCollider) isComponent() {}
