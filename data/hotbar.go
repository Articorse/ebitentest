package data

const (
	HotbarSlots             = 9
	HotbarMarginPx          = 5
	HotbarSlotGapPx         = 1
	HotbarStackCountSizePx  = 8
	HotbarStackCountInsetPx = 2
	InventoryRows           = 4
	InventoryColumns        = 9
	InventoryMarginPx       = 5
	// Between the inventory panel's edge and its slots, which is enough to clear its border
	InventoryPanelPaddingPx = 5
	// The inventory panel's image is cut into nine slices this many pixels across: corners kept as they are, edges
	// stretched along, and the middle stretched both ways
	InventoryPanelSlicePx = 3
	CornerPanelColumns    = 4
	TooltipTextSizePx     = 8
	TooltipPaddingPx      = 4
	// From the cursor to the tooltip's corner, so that the cursor doesn't hide its first letters
	TooltipOffsetPx = 8
	// From the top and right of the screen to the clock
	ClockMarginPx = 4
	// How far each area button's colour is inset from its slot's edges
	AreaButtonInsetPx = 8
	// The row under the inventory's slots showing what it carries against what it can
	InventoryWeightRowPx = 10
)
