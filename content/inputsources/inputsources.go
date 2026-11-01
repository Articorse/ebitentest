package inputsources

import (
	"ebittest/ecs"
	"encoding/gob"
	"fmt"
)

// Saved components refer to input sources by these values, so they must never change
const (
	InputType_Dummy  ecs.InputTypeEnum = 0
	InputType_Demo   ecs.InputTypeEnum = 1
	InputType_Follow ecs.InputTypeEnum = 2
	InputType_Human  ecs.InputTypeEnum = 3
	InputType_Loop   ecs.InputTypeEnum = 4
	InputType_Replay ecs.InputTypeEnum = 5
)

func Register(ecsContainer *ecs.ECSContainer) error {
	gob.Register(&InputDemoParams{})
	gob.Register(&InputFollowParams{})
	gob.Register(&InputLoopParams{})
	gob.Register(&InputReplayParams{})

	inputSources := []struct {
		id              ecs.InputTypeEnum
		inputSourceFunc ecs.InputSourceFunc
	}{
		{InputType_Dummy, DummyInputSource},
		{InputType_Demo, DemoInputSource},
		{InputType_Follow, FollowInputSource},
		{InputType_Human, HumanInputSource},
		{InputType_Loop, LoopInputSource},
		{InputType_Replay, ReplayInputSource},
	}

	for _, s := range inputSources {
		if err := ecsContainer.RegisterInputSource(s.id, s.inputSourceFunc); err != nil {
			return fmt.Errorf("error registering input source %v: %w", s.id, err)
		}
	}

	return nil
}
