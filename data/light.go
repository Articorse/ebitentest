package data

const (
	MaxLight = 255
	// Full daylight lets tiles be seen as far as vision reaches
	DaySkyLight = MaxLight
	// Seen from 12 tiles away
	NightSkyLight = 48
	// How far line of sight is followed, whatever the light. It's fixed, rather than the screen's edge, since vision
	// is part of the simulation and mustn't depend on the window or zoom.
	VisionMaxRangeTiles = 64
)
