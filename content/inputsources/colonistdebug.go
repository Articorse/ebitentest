package inputsources

import (
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
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
	Colonist_Roof:      "Roof",
}

var stopActionNames = map[StopAction]string{
	Stop_Build:   "Build",
	Stop_Chop:    "Chop",
	Stop_PickUp:  "Pick up",
	Stop_Deliver: "Deliver",
	Stop_PutDown: "Put down",
	Stop_MoveOff: "Move off",
	Stop_Roof:    "Roof",
}

var emergencyNames = map[EmergencyKind]string{
	Emergency_GetOut: "Getting out of the way of",
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
		if s.Amount > 0 {
			name, amount := "?", fmt.Sprint(s.Amount)
			if def, ok := ecsContainer.ItemAtlas().Def(s.Kind); ok {
				name, amount = def.Name(), def.AmountText(s.Amount)
			}
			line += fmt.Sprintf(" %s %s", amount, name)
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
	if s := p.InPassing; p.InPassingTick != 0 {
		name := "?"
		if def, ok := ecsContainer.ItemAtlas().Def(s.Kind); ok {
			name = def.Name()
		}
		lines = append(lines, fmt.Sprintf("In passing: %s %d %s at %d,%d on tick %d", stopActionNames[s.Action], s.Amount, name, s.Tile.X, s.Tile.Y, p.InPassingTick))
	}
	if em := p.Emergency; em.Kind != Emergency_None {
		line := fmt.Sprintf("Emergency: %s %d,%d", emergencyNames[em.Kind], em.Tile.X, em.Tile.Y)
		if em.Until != 0 {
			line += fmt.Sprintf(" until tick %d", em.Until)
		}
		lines = append(lines, line)
	}
	if j := p.Paused; j != nil {
		line := fmt.Sprintf("Paused: %s about %d,%d", colonistStateNames[j.State], j.Subject.X, j.Subject.Y)
		if j.Claimed {
			line += fmt.Sprintf(", claimed until tick %d", j.ClaimUntil)
		}
		lines = append(lines, line)
	}
	if p.Last != Colonist_Idle {
		lines = append(lines, fmt.Sprintf("Last: %s about %d,%d", colonistStateNames[p.Last], p.LastSubject.X, p.LastSubject.Y))
	}

	inv, ok := ecsContainer.Inventories.Get(e)
	if !ok {
		return lines, true
	}
	lines = append(lines, fmt.Sprintf("Carrying %s/%s", utils.FormatMass(ecsContainer.CarriedMass(e)), utils.FormatMass(inv.MaxMass())))
	for i := range inv.Capacity() {
		item, ok := inv.Slot(i)
		if !ok {
			continue
		}
		comp, ok := ecsContainer.Items.Get(item)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s %s", ecsContainer.ItemAmountText(comp), ecsContainer.ItemName(comp)))
	}
	return lines, true
}
