package data

// Saved entities refer to their faction by these values, so they must never change
type FactionEnum uint8

const (
	Faction_None FactionEnum = iota
	Faction_Colony
	Faction_Monsters
)

// Pairs of factions that are hostile to each other, either way round. Every other pair is neutral.
var HostileFactions = [][2]FactionEnum{
	{Faction_Colony, Faction_Monsters},
}
