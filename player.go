package main

import (
	"time"
)

type gamePlayer struct {
	x, y float64

	angle float64

	weapon *playerWeapon

	score int

	soulsRescued int

	health int

	holyWaters int

	garlicUntil    time.Time
	holyWaterUntil time.Time
}

func NewPlayer() (*gamePlayer, error) {
	p := &gamePlayer{
		weapon: &playerWeapon{
			sprite:   imageAtlas[ImageUzi],
			cooldown: 100 * time.Millisecond,
		},
		health: 3,
	}
	return p, nil
}
