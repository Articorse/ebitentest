package data

import "image/color"

var (
	// Premultiplied, and faint, since an area can cover much of the screen
	HomeAreaColor = color.RGBA{R: 20, G: 60, B: 20, A: 60}
	// Yellow-orange
	HarvestAreaColor = color.RGBA{R: 70, G: 45, B: 5, A: 70}
	// The rectangle being dragged out, marking or unmarking
	HomeMarkColor   = color.RGBA{R: 30, G: 110, B: 30, A: 110}
	AreaUnmarkColor = color.RGBA{R: 110, G: 25, B: 25, A: 110}
	// Yellow-orange
	HarvestMarkColor = color.RGBA{R: 120, G: 80, B: 10, A: 120}
	// The squares on the area menu's buttons, opaque so they read as the area's colour against the slot
	HomeButtonColor    = color.RGBA{R: 60, G: 170, B: 60, A: 255}
	HarvestButtonColor = color.RGBA{R: 230, G: 160, B: 30, A: 255}
	// Purple, shown, dragged out and on build mode's button
	StorageAreaColor   = color.RGBA{R: 50, G: 20, B: 70, A: 70}
	StorageMarkColor   = color.RGBA{R: 90, G: 40, B: 120, A: 120}
	StorageButtonColor = color.RGBA{R: 150, G: 70, B: 200, A: 255}
)

const (
	// The middle of the chunk at the world's origin, so that the colony's bounds, which are whole chunks, are centered
	// on it
	ColonyCenterTileX, ColonyCenterTileY = ChunkSize / 2, ChunkSize / 2
	// The chunks around the center's own that are the colony, so 1 is 3×3 chunks
	ColonyRadiusChunks = 1
	// How far around something built on home the home spreads, as a square
	HomeSpreadTiles = 15

	ColonistInventorySlots = 9
	// Under its carry weight along with its axe, so that it can pick things up from the start
	ColonistStartingWood   = 15
	ColonistMaxCarryWeight = 40
	// How often a colonist looks for work while it has none, since looking can mean searching the whole colony
	ColonistPlanEveryTicks = TPS
	// How many paths it looks for when choosing a job, best first, before settling for the best it's found
	ColonistPathSearches = 8
	// Chopping is done from next to a tree, not diagonally, where an axe swung from the far side of the tile can fall
	// short of it
	ColonistChopRangeTiles = 1
	// Spawned at each colony in a new world
	ColonistsPerColony = 5
	// Half the player's, so that what they're doing can be followed
	ColonistAcceleration = DefaultAcceleration / 2.0
	// How far an idle colonist standing on a site looks for somewhere off it
	ColonistStepOffRangeTiles = 3
	// How far around the first stack a colonist gathers others of its kind in the same go
	ColonistGatherRangeTiles = 5
)

// A job's score is its weight and bonuses less what walking to it costs, in the units paths are costed in, where a tile
// walked at full speed is 100
const (
	ColonistHaulWeight    = 3000
	ColonistBuildWeight   = 2500
	ColonistDeliverWeight = 2200
	ColonistFetchWeight   = 2000
	ColonistGatherWeight  = 1500
	ColonistChopWeight    = 1000
	ColonistStoreWeight   = 500
	// For the same kind of work as the last job
	ColonistSameWorkBonus = 300
	// For what naturally follows the last job, within ColonistFollowUpRangeTiles of what it was about
	ColonistFollowUpBonus      = 2000
	ColonistFollowUpRangeTiles = 5
	// How much of a path's cost comes off the score
	ColonistWalkCostPercent = 100
	// Off a site's score for each order still being worked on that was placed before its own, and for each site placed
	// before it in its own, each only up to a point
	ColonistLaterOrderCost    = 300
	ColonistLaterOrderMaxCost = 1500
	ColonistLaterSiteCost     = 30
	ColonistLaterSiteMaxCost  = 600
)

// What colonists gather where it lies loose on the harvest area
var HarvestItems = []ItemEnum{Item_Wood}
