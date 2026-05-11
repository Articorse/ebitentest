package ecscommon

import "github.com/hajimehoshi/ebiten/v2"

type PlayerConfig struct {
	Entity  Entity
	KeyMaps KeyMaps
}

// TODO: Associate with inputs, not players
type KeyMaps struct {
	Up    ebiten.Key
	Down  ebiten.Key
	Left  ebiten.Key
	Right ebiten.Key
}

type PlayerId string
