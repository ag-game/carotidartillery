package main

import (
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type gameCreep struct {
	x, y   float64
	sprite *ebiten.Image

	moveX, moveY float64

	tick       int
	nextAction int

	level *Level

	sync.Mutex
}

func NewCreep(sprite *ebiten.Image, level *Level) *gameCreep {
	return &gameCreep{
		x:      float64(1 + rand.Intn(64)),
		y:      float64(1 + rand.Intn(64)),
		sprite: sprite,
		level:  level,
	}
}

func (c *gameCreep) doNextAction() {
	c.moveX = (rand.Float64() - 0.5) / 10
	c.moveY = (rand.Float64() - 0.5) / 10

	c.nextAction = 400 + rand.Intn(1000)
}

func (c *gameCreep) Update() {
	c.Lock()
	defer c.Unlock()

	c.tick++
	if c.tick >= c.nextAction {
		c.doNextAction()
		c.tick = 0
	}

	c.x, c.y = c.level.Clamp(c.x+c.moveX, c.y+c.moveY)
}

func (c *gameCreep) Position() (float64, float64) {
	c.Lock()
	defer c.Unlock()
	return c.x, c.y
}
