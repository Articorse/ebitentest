package drawsystem

import (
	"ebittest/data"
	"ebittest/ecs"
	"ebittest/ecs/common"
	"ebittest/utils"
	"fmt"
	"strings"
)

// What's in it in all, such as "Holding 8.2kg O2, 1.8kg CO2"
func SealedAirText(sealed *ecs.SealedAir) string {
	air := sealed.Contents()
	return fmt.Sprintf("Holding %s O2, %s CO2", utils.FormatMass(int(air[data.Gas_Oxygen])), utils.FormatMass(int(air[data.Gas_CarbonDioxide])))
}

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
		temperature, err := ecsContainer.AirTemperatureOf(e)
		if err != nil {
			return "", err
		}
		lines = append(lines, "",
			fmt.Sprintf("Air %.1f C, %d g per tile", temperature.Celsius(), air.Total()/data.MilligramsPerGram),
			fmt.Sprintf("Breathing %d g O2, %d g CO2", air[data.Gas_Oxygen]/data.MilligramsPerGram, air[data.Gas_CarbonDioxide]/data.MilligramsPerGram))
		if !ecs.SafeAir(air) {
			lines = append(lines, "Bad air")
		}
		if sealed, ok := ecsContainer.WornSealedAir(e); ok {
			lines = append(lines, SealedAirText(sealed))
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
