package main

import (
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type creep struct {
	x, y   float64
	sprite *ebiten.Image

	moveX, moveY float64

	tick       int
	nextAction int

	sync.Mutex
}

func (c *creep) doNextAction() {
	c.moveX = (rand.Float64() - 0.5) / 10
	c.moveY = (rand.Float64() - 0.5) / 10

	c.nextAction = 400 + rand.Intn(1000)
}

func (c *creep) Update() {
	c.Lock()
	defer c.Unlock()

	c.tick++
	if c.tick >= c.nextAction {
		c.doNextAction()
		c.tick = 0
	}

	c.x, c.y = c.x+c.moveX, c.y+c.moveY
}

func (c *creep) Position() (float64, float64) {
	c.Lock()
	defer c.Unlock()
	return c.x, c.y
}
