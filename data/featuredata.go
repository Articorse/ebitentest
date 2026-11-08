package data

// Saved and generated worlds refer to features by these values, so they must never change
type FeatureEnum uint16

const (
	Feature_None FeatureEnum = iota
	Feature_Tree
	Feature_Stump
	Feature_Torch
	Feature_WoodDoor
	Feature_WoodDoorOpen
	Feature_WoodDoorVertical
	Feature_WoodDoorVerticalOpen
)
