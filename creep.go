package main

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	TypeVampire = iota
	TypeBat
	TypeGhost
)

type gameCreep struct {
	x, y float64

	sprites []*ebiten.Image

	frame     int
	frames    int
	lastFrame time.Time

	creepType int

	moveX, moveY float64

	tick       int
	nextAction int

	level  *Level
	player *gamePlayer

	health int

	sync.Mutex
}

func (c *gameCreep) queueNextAction() {
	if c.creepType == TypeBat {
		c.nextAction = 288 + rand.Intn(288)
		return
	}
	c.nextAction = 288 + rand.Intn(432)
}

func (c *gameCreep) runAway() {
	c.queueNextAction()

	randMovementA := (rand.Float64() - 0.5) / 8
	randMovementB := (rand.Float64() - 0.5) / 8

	c.moveX = c.x - c.player.x
	if c.moveX < 0 {
		c.moveX = math.Abs(randMovementA) * -1
	} else {
		c.moveX = math.Abs(randMovementA)
	}
	c.moveY = c.y - c.player.y
	if c.moveY < 0 {
		c.moveY = math.Abs(randMovementB) * -1
	} else {
		c.moveY = math.Abs(randMovementB)
	}
}

func (c *gameCreep) seekPlayer() {
	maxSpeed := 0.5 / 9
	minSpeed := 0.1 / 9

	a := angle(c.x, c.y, c.player.x, c.player.y)
	c.moveX = -math.Cos(a)
	c.moveY = -math.Sin(a)
	for {
		if (c.moveX < -minSpeed || c.moveX > minSpeed) || (c.moveY < -minSpeed || c.moveY > minSpeed) {
			break
		}

		c.moveX *= 1.1
		c.moveY *= 1.1
	}
	for {
		if c.moveX >= -maxSpeed && c.moveX <= maxSpeed && c.moveY >= -maxSpeed && c.moveY <= maxSpeed {
			break
		}

		c.moveX *= 0.9
		c.moveY *= 0.9
	}

	c.nextAction = 1440
}

func (c *gameCreep) doNextAction() {
	c.queueNextAction()

	randMovementA := (rand.Float64() - 0.5) / 12
	randMovementB := (rand.Float64() - 0.5) / 12

	repelled := c.repelled()
	if !repelled && rand.Intn(13) == 0 {
		c.seekPlayer()
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
}

func (c *gameCreep) repelled() bool {
	repelled := !c.player.garlicUntil.IsZero() || !c.player.holyWaterUntil.IsZero()
	return repelled
}

func (c *gameCreep) Update() {
	c.Lock()
	defer c.Unlock()

	if c.health == 0 {
		return
	}

	c.tick++

	repelled := c.repelled()

	dx, dy := deltaXY(c.x, c.y, c.player.x, c.player.y)
	seekDistance := 2.0
	if !repelled && dx < seekDistance && dy < seekDistance {
		c.queueNextAction()
		c.seekPlayer()
	} else if c.tick >= c.nextAction {
		c.doNextAction()
		c.tick = 0
	}

	x, y := c.x+c.moveX, c.y+c.moveY
	if !c.level.isFloor(x, y, false) {
		c.nextAction = 0
		return
	}
	c.x, c.y = x, y

	if repelled {
		dx, dy := deltaXY(c.x, c.y, c.player.x, c.player.y)
		if dx <= 3 && dy <= 3 {
			c.runAway()
		}
	}

	for _, item := range c.level.items {
		if item.health == 0 {
			continue
		}

		dx, dy := deltaXY(c.x, c.y, item.x, item.y)
		if dx <= 2 && dy <= 2 {
			c.runAway()
		}
	}
}

func (c *gameCreep) Position() (float64, float64) {
	c.Lock()
	defer c.Unlock()
	return c.x, c.y
}

func (c *gameCreep) killScore() int {
	switch c.creepType {
	case TypeVampire:
		return 50
	case TypeBat:
		return 125
	case TypeGhost:
		return 75
	default:
		return 0
	}
}
