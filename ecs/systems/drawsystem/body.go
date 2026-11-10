package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"fmt"
	"strings"
)

// Each part's health, then each capacity's efficiency in whole percent
func BodyStatusText(e common.EntityId, ecsContainer *ecs.ECSContainer) (string, error) {
	parts, capacities, err := ecsContainer.BodyStatus(e)
	if err != nil {
		return "", err
	}

	lines := []string{}
	for _, p := range parts {
		lines = append(lines, fmt.Sprintf("%s %d/%d", p.Name, p.Health, p.MaxHealth))
	}
	if len(capacities) > 0 {
		lines = append(lines, "")
	}
	for _, c := range capacities {
		lines = append(lines, fmt.Sprintf("%s %d%%", c.Name, c.Efficiency*100/data.FullEfficiency))
	}
	if air, ok := ecsContainer.AirOf(e); ok && ecsContainer.OxygenPerSecond(e) > 0 {
		pos, err := ecsContainer.WorldPos(e)
		if err != nil {
			return "", err
		}
		temperature := ecsContainer.Terrain().AirTemperature(ecsContainer.AirTile(pos))
		lines = append(lines, "",
			fmt.Sprintf("Air %.1f C, %d g per tile", temperature.Celsius(), air.Total()/data.MilligramsPerGram),
			fmt.Sprintf("Breathing %d g O2, %d g CO2", air[data.Gas_Oxygen]/data.MilligramsPerGram, air[data.Gas_CarbonDioxide]/data.MilligramsPerGram))
		if !ecs.SafeAir(air) {
			lines = append(lines, "Bad air")
		}
	}
	if conditions := ecsContainer.Conditions(e); len(conditions) > 0 {
		lines = append(lines, "")
		for _, c := range conditions {
			line := fmt.Sprintf("%s %d%% (%s)", c.Name, c.Percent, c.Part)
			if c.Worsening {
				line += ", worsening"
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}
