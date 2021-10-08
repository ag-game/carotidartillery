package main

import (
	"math"
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

	level  *Level
	player *gamePlayer

	health int

	killScore int

	sync.Mutex
}

func (c *gameCreep) doNextAction() {
	randMovementA := (rand.Float64() - 0.5) / 7
	randMovementB := (rand.Float64() - 0.5) / 7
	if rand.Intn(7) == 0 {
		// Seek player.
		c.moveX = c.x - c.player.x
		if c.moveX < 0 {
			c.moveX = math.Abs(randMovementA)
		} else {
			c.moveX = math.Abs(randMovementA) * -1
		}
		c.moveY = c.y - c.player.y
		if c.moveY < 0 {
			c.moveY = math.Abs(randMovementB)
		} else {
			c.moveY = math.Abs(randMovementB) * -1
		}
	} else {
		c.moveX = randMovementA
		c.moveY = randMovementB
	}

	if c.x <= 2 && c.moveX < 0 {
		c.moveX *= 1
	} else if c.x >= float64(c.level.w-3) && c.moveX > 0 {
		c.moveX *= 1
	}
	if c.y <= 2 && c.moveY > 0 {
		c.moveY *= 1
	} else if c.y >= float64(c.level.h-3) && c.moveY < 0 {
		c.moveY *= 1
	}

	c.nextAction = 144 + rand.Intn(720)
}

func NewCreep(sprite *ebiten.Image, level *Level, player *gamePlayer) *gameCreep {
	return &gameCreep{
		x:         float64(1 + rand.Intn(108)),
		y:         float64(1 + rand.Intn(108)),
		sprite:    sprite,
		level:     level,
		player:    player,
		health:    1,
		killScore: 50,
	}
}

func (c *gameCreep) Update() {
	c.Lock()
	defer c.Unlock()

	if c.health == 0 {
		return
	}

	c.tick++
	if c.tick >= c.nextAction {
		c.doNextAction()
		c.tick = 0
	}

	x, y := c.x+c.moveX, c.y+c.moveY
	clampX, clampY := c.level.Clamp(x, y)
	c.x, c.y = clampX, clampY
	if clampX != x || clampY != y {
		c.nextAction = 0
	}
}

func (c *gameCreep) Position() (float64, float64) {
	c.Lock()
	defer c.Unlock()
	return c.x, c.y
}
