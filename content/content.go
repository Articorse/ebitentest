package content

import (
	"ebittest/content/abilities"
	"ebittest/content/inputsources"
	"ebittest/content/items"
	"ebittest/content/timerfuncs"
	"ebittest/ecs"
)

func Register(ecsContainer *ecs.ECSContainer) error {
	if err := abilities.Register(ecsContainer); err != nil {
		return err
	}
	if err := timerfuncs.Register(ecsContainer); err != nil {
		return err
	}
	if err := inputsources.Register(ecsContainer); err != nil {
		return err
	}
	if err := items.Register(ecsContainer); err != nil {
		return err
	}

	return nil
}
