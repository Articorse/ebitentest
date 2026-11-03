package data

const (
	MaxLight = 255
	// The darkest that anything lit is drawn, as a share of full brightness
	MinLightBrightness = 0.01
	// Full daylight lets tiles be seen as far as vision reaches
	DaySkyLight = MaxLight
	// Seen from 12 tiles away
	NightSkyLight = 48
	// What light loses with each tile it spreads, so daylight reaches 7 tiles under a roof and night sky light 1
	LightFalloffPerTile = 32
	// How far the player sees in full light with unhurt eyes, whatever the light. It's fixed, rather than the screen's
	// edge, since vision is part of the simulation and mustn't depend on the window or zoom.
	PlayerVisionRangeTiles = 64
)
