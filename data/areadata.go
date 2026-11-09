package data

// The areas marked out on tiles, for colonists to know where to work and what to do there
type AreaEnum uint8

const (
	Area_None AreaEnum = iota
	Area_Home
	Area_Harvest
	// Where what's put away lies, marked out in build mode rather than the area menu, since it's part of what the colony
	// is made of
	Area_Storage
	// Where colonists build roofs, and where they take built ones down, which a tile can only be one of
	Area_Roof
	Area_NoRoof
	AreaCount
)

// The area that marking a tile as this one takes it off
func OppositeArea(a AreaEnum) (AreaEnum, bool) {
	switch a {
	case Area_Roof:
		return Area_NoRoof, true
	case Area_NoRoof:
		return Area_Roof, true
	}
	return Area_None, false
}
