package main

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	itemTypeGarlic = iota
	itemTypeHolyWater
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
	switch item.itemType {
	case itemTypeGarlic:
		return 275
	case itemTypeHolyWater:
		return 150
	default:
		return 0
	}
}
