package ecscommon

import "fmt"

type ErrorMissingComponentDependency struct {
	Entity           EntityId
	PresentComponent string
	MissingComponent string
}

type ErrorMissingExpectedComponent struct {
	Entity           EntityId
	MissingComponent string
}

func (x *ErrorMissingComponentDependency) Error() string {
	return fmt.Sprintf("entity %d had Component %s, but no Component %s\n", x.Entity, x.PresentComponent, x.MissingComponent)
}

func (x *ErrorMissingExpectedComponent) Error() string {
	return fmt.Sprintf("entity %d missing expected Component %s\n", x.Entity, x.MissingComponent)
}
