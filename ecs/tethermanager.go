package ecs

import (
	"ebittest/ecs/common"
	"fmt"
)

type tetherManager struct{}

func NewTetherComponent(anchor common.EntityId) *tether {
	return &tether{
		anchor: anchor,
	}
}

func (tetherManager) GetAnchor(e common.EntityId, ecsContainer *ECSContainer) (common.EntityId, error) {
	tetherComp, err := ecsContainer.Tethers.getComponent(e)
	if err != nil {
		return -1, fmt.Errorf("could not get tether component of entity %d: %v", e, err)
	}

	return tetherComp.anchor, nil
}

func (tetherManager) SetAnchor(e common.EntityId, anchor common.EntityId, ecsContainer *ECSContainer) error {
	tetherComp, err := ecsContainer.Tethers.getComponent(e)
	if err != nil {
		return fmt.Errorf("could not get tether component of entity %d: %v", e, err)
	}

	tetherComp.anchor = anchor
	return nil
}
