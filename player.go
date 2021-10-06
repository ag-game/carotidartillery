package main

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type gamePlayer struct {
	x, y float64

	angle float64

	weapon *playerWeapon

	health int
}

func NewPlayer() (*gamePlayer, error) {
	f, err := assetsFS.Open("assets/weapons/uzi.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	uziSprite := ebiten.NewImageFromImage(img)

	p := &gamePlayer{
		weapon: &playerWeapon{
			sprite:   uziSprite,
			cooldown: 100 * time.Millisecond,
		},
		health: 7,
	}
	return p, nil
}
