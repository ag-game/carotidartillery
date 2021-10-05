package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type playerWeapon struct {
	sprite        *ebiten.Image
	spriteFlipped *ebiten.Image
	lastFire      time.Time
	cooldown      time.Duration
}
