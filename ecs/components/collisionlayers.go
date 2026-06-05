package components

type LayerMask uint16

const (
	Layer_Player             = LayerMask(0b1000000000000000)
	Layer_Enemy              = LayerMask(0b0100000000000000)
	Layer_FriendlyProjectile = LayerMask(0b0010000000000000)
	Layer_EnemyProjectile    = LayerMask(0b0001000000000000)
	Layer_Terrain            = LayerMask(0b0000100000000000)
	Layer_Platform           = LayerMask(0b0000010000000000)
)

type CollisionLayer struct {
	layers LayerMask
	mask   LayerMask
}

func (CollisionLayer) isComponent() {}

func (x CollisionLayer) Copy() CollisionLayer {
	return CollisionLayer{
		layers: x.layers,
		mask:   x.mask,
	}
}
