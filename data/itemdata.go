package data

// Generic items, which are defined by the item atlas. Saved items and drop tables refer to them by these values, so
// they must never change.
type ItemEnum uint16

const (
	// Unique items, such as tools, which are each their own entity and never stack
	Item_None ItemEnum = iota
	Item_Wood
	Item_Torch
	// Gases dropped out of the air, which they go back into once it's warm enough
	Item_Oxygen
	Item_CarbonDioxide
)
