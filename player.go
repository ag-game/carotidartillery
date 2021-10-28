package main

import (
	"time"
)

var weaponUzi = &playerWeapon{
	sprite:   imageAtlas[ImageUzi],
	cooldown: 100 * time.Millisecond,
}

type gamePlayer struct {
	x, y float64

	angle float64

	weapon *playerWeapon

	hasTorch bool

	score int

	soulsRescued int

	health int

	holyWaters int

	garlicUntil    time.Time
	holyWaterUntil time.Time
}

func NewPlayer() (*gamePlayer, error) {
	p := &gamePlayer{
		weapon:   weaponUzi,
		hasTorch: true,
		health:   3,
	}
	return p, nil
}
