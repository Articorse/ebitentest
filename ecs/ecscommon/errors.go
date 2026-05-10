package ecscommon

import "fmt"

type ErrorMissingComponent struct {
	Entity           Entity
	PresentComponent string
	MissingComponent string
}

func (x *ErrorMissingComponent) Error() string {
	return fmt.Sprintf("entity %d had Component %s, but no Component %s\n", x.Entity, x.PresentComponent, x.MissingComponent)
}
