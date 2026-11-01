package ecs

import "fmt"

// Saved components refer to abilities, timer functions and input sources by ID, so an ID that's taken twice is refused
// rather than silently changing what saves refer to
func (x *ECSContainer) RegisterAbility(id AbilityEnum, abilityFunc AbilityFunc) error {
	if id == Ability_None {
		return fmt.Errorf("cannot register an ability as none")
	}
	if _, ok := x.abilityFuncs[id]; ok {
		return fmt.Errorf("ability %v is already registered", id)
	}

	x.abilityFuncs[id] = abilityFunc
	return nil
}

func (x *ECSContainer) RegisterTimerFunc(id TimerFuncEnum, timerFunc TimerFunc) error {
	if id == TimerFunc_None {
		return fmt.Errorf("cannot register a timer function as none")
	}
	if _, ok := x.timerFuncs[id]; ok {
		return fmt.Errorf("timer function %v is already registered", id)
	}

	x.timerFuncs[id] = timerFunc
	return nil
}

func (x *ECSContainer) RegisterInputSource(id InputTypeEnum, inputSourceFunc InputSourceFunc) error {
	if _, ok := x.inputSourceFuncs[id]; ok {
		return fmt.Errorf("input source %v is already registered", id)
	}

	x.inputSourceFuncs[id] = inputSourceFunc
	return nil
}
