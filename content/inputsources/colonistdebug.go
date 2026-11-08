package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
)

var colonistStateNames = map[ColonistState]string{
	Colonist_Idle:      "Idle",
	Colonist_Construct: "Build",
	Colonist_Deliver:   "Deliver",
	Colonist_Chop:      "Chop",
	Colonist_Gather:    "Gather",
	Colonist_Store:     "Store",
	Colonist_Fetch:     "Fetch",
	Colonist_Haul:      "Haul",
}

var stopActionNames = map[StopAction]string{
	Stop_Build:   "Build",
	Stop_Chop:    "Chop",
	Stop_PickUp:  "Pick up",
	Stop_Deliver: "Deliver",
	Stop_PutDown: "Put down",
	Stop_MoveOff: "Move off",
}

// Its job and what it's carrying, for the debug overlay. False for anything that isn't a colonist.
func ColonistDebugLines(e common.EntityId, ecsContainer *ecs.ECSContainer) ([]string, bool) {
	in, ok := ecsContainer.Inputs.Get(e)
	if !ok {
		return nil, false
	}
	p, ok := in.Params().(*InputColonistParams)
	if !ok || p == nil {
		return nil, false
	}

	lines := []string{fmt.Sprintf("Colonist %d: %s", e, colonistStateNames[p.State])}
	for i, s := range p.Stops {
		line := "  " + stopActionNames[s.Action]
		if s.Count > 0 {
			name := "?"
			if def, ok := ecsContainer.ItemAtlas().Def(s.Kind); ok {
				name = def.Name()
			}
			line += fmt.Sprintf(" %d %s", s.Count, name)
		}
		line += fmt.Sprintf(" at %d,%d", s.Tile.X, s.Tile.Y)
		if i == 0 {
			line = ">" + line[1:]
			if p.Acted {
				line += " (done)"
			}
		}
		lines = append(lines, line)
	}
	if p.State != Colonist_Idle {
		lines = append(lines, fmt.Sprintf("Scored %d, about %d,%d", p.Score, p.Subject.X, p.Subject.Y))
		lines = append(lines, fmt.Sprintf("Standing at %d,%d, %d tiles of path", p.Stand.X, p.Stand.Y, max(len(p.path)-1, 0)))
	} else {
		lines = append(lines, fmt.Sprintf("Next looks for work on tick %d", p.NextPlanTick))
	}
	if p.Last != Colonist_Idle {
		lines = append(lines, fmt.Sprintf("Last: %s about %d,%d", colonistStateNames[p.Last], p.LastSubject.X, p.LastSubject.Y))
	}

	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return lines, true
	}
	lines = append(lines, fmt.Sprintf("Carrying %d/%d", ecsContainer.CarriedWeight(e), inv.MaxWeight()))
	for i := range inv.Capacity() {
		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		comp, ok := ecsContainer.Items.Get(item)
		if !ok {
			continue
		}
		line := "  " + ecsContainer.ItemName(comp)
		if comp.Count() > 1 {
			line = fmt.Sprintf("  %d %s", comp.Count(), ecsContainer.ItemName(comp))
		}
		lines = append(lines, line)
	}
	return lines, true
}
