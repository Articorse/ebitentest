package climate

import (
	"ebittest/data"
	"encoding/json"
	"fmt"
	"os"
)

// Milligrams of each gas, by its id, in whole numbers so that air flows the same on every machine
type Air [data.GasCount]int64

func (a Air) Total() int64 {
	total := int64(0)
	for _, mg := range a {
		total += mg
	}
	return total
}

type Gas struct {
	Id   data.GasEnum
	Name string
}

type gasDto struct {
	Id   uint16 `json:"id"`
	Name string `json:"name"`
}

func LoadGasesFromJson(path string) (map[data.GasEnum]Gas, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading gases json file: %w", err)
	}
	var dtos []gasDto
	if err := json.Unmarshal(f, &dtos); err != nil {
		return nil, fmt.Errorf("error unmarshalling gases json: %w", err)
	}
	gases := map[data.GasEnum]Gas{}
	for _, d := range dtos {
		id := data.GasEnum(d.Id)
		switch _, taken := gases[id]; {
		case id == data.Gas_None:
			return nil, fmt.Errorf("gas %q has id 0, which means none", d.Name)
		case id >= data.GasCount:
			return nil, fmt.Errorf("gas %q has id %d, past the last there's room for", d.Name, d.Id)
		case taken:
			return nil, fmt.Errorf("gas id %d is defined twice", d.Id)
		}
		gases[id] = Gas{Id: id, Name: d.Name}
	}
	return gases, nil
}
