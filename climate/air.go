package climate

import (
	"ebittest/data"
	"encoding/json"
	"fmt"
	"math"
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
	// Below it, it drops out of the air as its item, which turns back into it above it
	Condense Centicelsius
	Item     data.ItemEnum
	// In mJ per gram per degree
	SpecificHeat int64
	// In mW per degree across an edge between tiles, with a kilogram of it on the tile, and in proportion to how much
	// of it there is, so that thin air barely passes heat on and none passes none
	Conductance int64
}

type gasDto struct {
	Id              uint16  `json:"id"`
	Name            string  `json:"name"`
	CondenseCelsius float64 `json:"condenseCelsius"`
	Item            uint16  `json:"item"`
	// In J per gram per degree
	SpecificHeat float64 `json:"specificHeat"`
	// In W per degree
	Conductance float64 `json:"conductance"`
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
		case d.Item == uint16(data.Item_None):
			return nil, fmt.Errorf("gas %d has no item to drop out of the air as", d.Id)
		case d.SpecificHeat <= 0 || d.Conductance < 0:
			return nil, fmt.Errorf("gas %d takes %v J per gram per degree and passes %v W per degree on, want the first positive and the second not negative", d.Id, d.SpecificHeat, d.Conductance)
		}
		gases[id] = Gas{
			Id: id, Name: d.Name, Condense: Centicelsius(math.Round(d.CondenseCelsius * 100)),
			Item:         data.ItemEnum(d.Item),
			SpecificHeat: int64(math.Round(d.SpecificHeat * 1000)),
			Conductance:  int64(math.Round(d.Conductance * 1000)),
		}
	}
	return gases, nil
}
