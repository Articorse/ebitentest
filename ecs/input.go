package ecs

import (
	"ebittest/data"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"maps"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The IDs are defined by the content that registers them
type InputTypeEnum uint8

type InputType uint64

const (
	Input_Analog1Y InputType = iota
	Input_Analog1X
	Input_Analog2Y
	Input_Analog2X
	Input_MainHandAbility1
	Input_MainHandAbility2
	Input_OffHandAbility1
	Input_OffHandAbility2
	Input_Ability1
	Input_Ability2
	// One per hotbar slot, in order
	Input_SelectSlot1
	Input_SelectSlot2
	Input_SelectSlot3
	Input_SelectSlot4
	Input_SelectSlot5
	Input_SelectSlot6
	Input_SelectSlot7
	Input_SelectSlot8
	Input_SelectSlot9
	Input_DropItem
	// Held to do all of something at once, such as delivering every piece of a material that's needed
	Input_All
)

type InputSourceFunc func(
	e common.EntityId,
	tick uint64,
	inputParams InputParams,
	ecsContainer *ECSContainer,
) (InputState, error)

type InputParams interface {
	IsInputParams()
	Copy() InputParams
}

type InputKey struct {
	GamepadId   ebiten.GamepadID
	KeyboardKey [2]ebiten.Key
	MouseKey    [2]ebiten.MouseButton
	GamepadKey  [2]ebiten.StandardGamepadButton
	GamepadAxis ebiten.StandardGamepadAxis
}

func NewKeyboardInputKey(valueUp, valueDown ebiten.Key) InputKey {
	k := newUnsetInputKey()
	k.KeyboardKey = [2]ebiten.Key{valueUp, valueDown}
	return k
}

func NewMouseInputKey(valueUp, valueDown ebiten.MouseButton) InputKey {
	k := newUnsetInputKey()
	k.MouseKey = [2]ebiten.MouseButton{valueUp, valueDown}
	return k
}

func NewGamepadButtonInputKey(gamepadId ebiten.GamepadID, valueUp, valueDown ebiten.StandardGamepadButton) InputKey {
	k := newUnsetInputKey()
	k.GamepadId = gamepadId
	k.GamepadKey = [2]ebiten.StandardGamepadButton{valueUp, valueDown}
	return k
}

func NewGamepadAxisInputKey(gamepadId ebiten.GamepadID, axis ebiten.StandardGamepadAxis) InputKey {
	k := newUnsetInputKey()
	k.GamepadId = gamepadId
	k.GamepadAxis = axis
	return k
}

func (x InputKey) GetInput() float64 {
	if x.KeyboardKey[0] != -1 {
		if ebiten.IsKeyPressed(x.KeyboardKey[0]) {
			return 1
		}
	}
	if x.KeyboardKey[1] != -1 {
		if ebiten.IsKeyPressed(x.KeyboardKey[1]) {
			return -1
		}
	}
	if x.MouseKey[0] != -1 {
		if ebiten.IsMouseButtonPressed(x.MouseKey[0]) {
			return 1
		}
	}
	if x.MouseKey[1] != -1 {
		if ebiten.IsMouseButtonPressed(x.MouseKey[1]) {
			return -1
		}
	}
	if x.GamepadKey[0] != -1 {
		if ebiten.IsStandardGamepadButtonPressed(x.GamepadId, x.GamepadKey[0]) {
			return 1
		}
	}
	if x.GamepadKey[1] != -1 {
		if ebiten.IsStandardGamepadButtonPressed(x.GamepadId, x.GamepadKey[1]) {
			return -1
		}
	}
	if x.GamepadAxis != -1 {
		axisVal := ebiten.StandardGamepadAxisValue(x.GamepadId, x.GamepadAxis)
		if math.Abs(axisVal) > data.GamepadDeadzone {
			return axisVal
		} else {
			return 0
		}
	}

	return 0
}

type Input struct {
	config        map[InputType]InputKey
	inputType     InputTypeEnum
	facingInput   FacingInputEnum
	params        InputParams
	lastFacingDir utils.Vec2f
	// Not saved: a key held through loading only counts as pressed once more
	held map[InputType]bool
	// Wheel movement short of a whole step, so that a touchpad's small scrolls add up rather than each counting as one.
	// Forgotten when the wheel turns the other way, so that a full step back always steps back.
	wheel float64
}

func newUnsetInputKey() InputKey {
	return InputKey{
		KeyboardKey: [2]ebiten.Key{-1, -1},
		MouseKey:    [2]ebiten.MouseButton{-1, -1},
		GamepadKey:  [2]ebiten.StandardGamepadButton{-1, -1},
		GamepadAxis: -1,
	}
}

type FacingInputEnum uint8

const (
	Facing_None FacingInputEnum = iota
	Facing_Mouse
	Facing_Analog2
)

type InputState struct {
	Analog1X, Analog1Y float64
	Analog2X, Analog2Y float64
	MainHandEqAbility1 float64
	MainHandEqAbility2 float64
	OffHandEqAbility1  float64
	OffHandEqAbility2  float64
	Ability1           float64
	Ability2           float64
	AimPos             utils.Vec2f
	// Without it, AimPos means nothing, so that aiming at the world's origin isn't mistaken for not aiming
	HasAim bool
	// 1-based, so that the zero value selects nothing
	SelectSlot int
	// Steps the selected hotbar slot this far down, wrapping around, when SelectSlot doesn't select one
	ScrollSlot int
	DropItem   bool
	All        bool
	// What a click in an inventory menu changed. Menus send these rather than changing the inventory themselves, so
	// that the change happens in the simulation's tick like any other input.
	Inventory InventoryAction
	// Blueprints put down, each wherever it fits
	PlaceBlueprint BlueprintPlacement
	// A blueprint taken away, which only works while nothing has been delivered to it
	RemoveBlueprint BlueprintRemoval
	// The rectangle of tiles put in an area, or taken out of it
	Area AreaMarking
	// What lies on a tile within reach picked up, as much as there's room for
	PickUp ItemPickUp
	// Storage marked out on the tiles, or taken off them
	Storage StorageMarking
	// Some of what's in an inventory slot put down on a tile within reach, and only it
	PutDown ItemPutDown
	// What lies on a tile within reach moved onto another within reach, without being carried
	MoveOnGround GroundMove
	// The roof over a tile within the constructor's reach built or taken down, which colonists do on their areas
	Roof RoofWork
}

type RoofWork struct {
	Work bool
	Tile utils.Vec2i
}

type GroundMove struct {
	Move     bool
	From, To utils.Vec2i
}

// Tiles rather than a rectangle, since only the UI knows which tiles have been seen
type StorageMarking struct {
	Mark, Erase bool
	Tiles       []utils.Vec2i
}

type ItemPickUp struct {
	PickUp bool
	Tile   utils.Vec2i
	// Up to how much, or all there is with none
	Amount int
}

type ItemPutDown struct {
	PutDown bool
	From    int
	Amount  int
	Tile    utils.Vec2i
}

// No area is no marking
type AreaMarking struct {
	Area        data.AreaEnum
	Erase       bool
	First, Last utils.Vec2i
}

type BlueprintRemoval struct {
	Remove bool
	Tile   utils.Vec2i
}

type BlueprintPlacement struct {
	// None when nothing is put down
	Kind  data.BlueprintEnum
	Tiles []utils.Vec2i
}

type InventoryActionKind uint8

const (
	InventoryAction_None InventoryActionKind = iota
	// Moves the item in inventory slot From to slot To
	InventoryAction_Move
	// Points hotbar slot To at the item in inventory slot From
	InventoryAction_Bind
	// Points hotbar slot To at nothing
	InventoryAction_Unbind
)

type InventoryAction struct {
	Kind     InventoryActionKind
	From, To int
}

func NewInputComponent(config map[InputType]InputKey, inputType InputTypeEnum, params InputParams, facingInput FacingInputEnum) *Input {
	return &Input{config: config, inputType: inputType, params: params, facingInput: facingInput}
}

func (x *Input) Value(inputType InputType) float64 {
	inKey, ok := x.config[inputType]
	if !ok {
		return 0
	}

	return inKey.GetInput()
}

func (x *Input) Params() InputParams {
	return x.params
}

func (x *Input) Type() InputTypeEnum {
	return x.inputType
}

// True only on the tick the input goes from released to held, so that holding it down acts once
func (x *Input) JustPressed(inputType InputType) bool {
	if x.held == nil {
		x.held = make(map[InputType]bool)
	}

	held := x.Value(inputType) > 0
	wasHeld := x.held[inputType]
	x.held[inputType] = held
	return held && !wasHeld
}

// Adds how far the wheel moved down, which is read every frame, since a frame doesn't always run a tick
func (x *Input) AddWheel(down float64) {
	if down*x.wheel < 0 {
		x.wheel = 0
	}
	x.wheel += down
}

// Whole steps scrolled down since the last call; negative is up
func (x *Input) WheelSteps() int {
	steps := int(x.wheel)
	x.wheel -= float64(steps)
	return steps
}

func (x *Input) FacingInput() FacingInputEnum {
	return x.facingInput
}

func (x *Input) LastFacingDir() utils.Vec2f {
	return x.lastFacingDir
}

func (x *Input) SetLastFacingDir(facingDir utils.Vec2f) {
	x.lastFacingDir = facingDir
}

func (x *ECSContainer) InputSourceFunc(inputType InputTypeEnum) (InputSourceFunc, error) {
	inputSourceFunc, ok := x.inputSourceFuncs[inputType]
	if !ok {
		return nil, fmt.Errorf("input source function for input type %v not found", inputType)
	}

	return inputSourceFunc, nil
}

func (x Input) Copy() Input {
	return Input{
		config:        x.config,
		inputType:     x.inputType,
		facingInput:   x.facingInput,
		params:        copyInputParams(x.params),
		lastFacingDir: utils.Vec2f{},
	}
}

func (x Input) addTo(ecsContainer *ECSContainer, e common.EntityId) {
	ecsContainer.Inputs.addComponent(e, x.Copy())
}

type inputDto struct {
	Config        map[InputType]InputKey
	InputType     InputTypeEnum
	FacingInput   FacingInputEnum
	Params        InputParams
	LastFacingDir utils.Vec2f
}

func (x Input) toDto() (ComponentDto, error) {
	return &inputDto{
		Config:        maps.Clone(x.config),
		InputType:     x.inputType,
		FacingInput:   x.facingInput,
		Params:        copyInputParams(x.params),
		LastFacingDir: x.lastFacingDir,
	}, nil
}

func copyInputParams(params InputParams) InputParams {
	if params == nil {
		return nil
	}
	return params.Copy()
}

func (x *inputDto) toComponent() (Component, error) {
	return &Input{
		config:        x.Config,
		inputType:     x.InputType,
		facingInput:   x.FacingInput,
		params:        x.Params,
		lastFacingDir: x.LastFacingDir,
	}, nil
}
