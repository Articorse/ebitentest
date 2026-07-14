package ecs

type persistentManager struct{}

func NewPersistentComponent() *persistent {
	return &persistent{}
}
