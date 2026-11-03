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
	return strings.Join(lines, "\n"), nil
}
