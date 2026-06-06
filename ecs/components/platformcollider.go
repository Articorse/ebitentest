package components

type PlatformCollider struct {
	BaseCollider
}

func (x *PlatformCollider) getBaseCollider() *BaseCollider { return &x.BaseCollider }

func (PlatformCollider) isComponent() {}
