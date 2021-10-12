package main

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	itemTypeGarlic = iota
)

type gameItem struct {
	x, y float64

	sprite *ebiten.Image

	itemType int

	level  *Level
	player *gamePlayer

	health int

	sync.Mutex
}

func (item *gameItem) useScore() int {
	return 275
}
