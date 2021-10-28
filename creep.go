package main

import (
	"log"
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
	TypeTorch
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

	angle float64

	sync.Mutex
}

func newCreep(creepType int, l *Level, p *gamePlayer) *gameCreep {
	sprites := []*ebiten.Image{
		imageAtlas[ImageVampire1],
		imageAtlas[ImageVampire2],
		imageAtlas[ImageVampire3],
		imageAtlas[ImageVampire2],
	}
	startingHealth := 1
	if creepType == TypeBat {
		sprites = []*ebiten.Image{
			batSS.Frame1,
			batSS.Frame2,
			batSS.Frame3,
			batSS.Frame4,
			batSS.Frame5,
			batSS.Frame6,
			batSS.Frame7,
		}
		startingHealth = 2
	} else if creepType == TypeGhost {
		sprites = []*ebiten.Image{
			imageAtlas[ImageGhost1],
		}
		startingHealth = 1
	} else if creepType == TypeTorch {
		sprites = []*ebiten.Image{
			sandstoneSS.TorchTop1,
			sandstoneSS.TorchTop2,
			sandstoneSS.TorchTop3,
			sandstoneSS.TorchTop4,
			sandstoneSS.TorchTop5,
			sandstoneSS.TorchTop6,
			sandstoneSS.TorchTop7,
			sandstoneSS.TorchTop8,
		}
	}

	startingFrame := 0
	if len(sprites) > 1 {
		startingFrame = rand.Intn(len(sprites))
	}

	var x, y float64
	if creepType != TypeTorch {
		x, y = l.newSpawnLocation()
	}

	return &gameCreep{
		creepType: creepType,
		x:         x,
		y:         y,
		sprites:   sprites,
		frames:    len(sprites),
		frame:     startingFrame,
		level:     l,
		player:    p,
		health:    startingHealth,
	}
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

	if c.creepType == TypeGhost {
		c.angle = angle(c.x, c.y, c.player.x, c.player.y)

		// TODO optimize
		if c.angle > math.Pi/2 || c.angle < -1*math.Pi/2 {
			c.sprites = []*ebiten.Image{
				imageAtlas[ImageGhost1R],
			}
			c.angle = c.angle - math.Pi
		} else {
			c.sprites = []*ebiten.Image{
				imageAtlas[ImageGhost1],
			}
		}
	}

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

// TODO return true when creep is facing player and player is facing creep
func (c *gameCreep) facingPlayer() bool {
	mod := func(v float64) float64 {
		for v > math.Pi {
			v -= math.Pi
		}
		for v < math.Pi*-1 {
			v += math.Pi
		}
		return v
	}
	_ = mod

	ca := math.Remainder(c.angle, math.Pi)
	ca = math.Remainder(ca, -math.Pi)
	if ca < 0 {
		ca = math.Pi + ca
	}
	ca = c.angle
	if ca < 0 {
		ca = math.Pi*2 + ca
	}
	pa := math.Remainder(c.player.angle, math.Pi)
	pa = math.Remainder(pa, -math.Pi)
	if pa < 0 {
		pa = math.Pi + pa
	}
	pa = c.player.angle
	if pa < 0 {
		pa = math.Pi*2 + pa
	}

	a := ca - pa
	if pa > ca {
		a = pa - ca
	}

	if rand.Intn(70) == 0 {
		// TODO
		log.Println(ca, pa, a)
	}

	return a < 2
	//a2 := c.player.angle - c.angle
	// TODO
	//return a > math.Pi/2*-1 && a2 > math.Pi/2*-1 && a < math.Pi/2 && a2 < math.Pi/2
}

func (c *gameCreep) Update() {
	c.Lock()
	defer c.Unlock()

	if c.health == 0 {
		return
	}

	if c.creepType == TypeTorch {
		return
	}

	c.tick++

	repelled := c.repelled()

	if c.creepType == TypeGhost && c.facingPlayer() {
		return
	}

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
	if c.level.isFloor(x, y) {
		c.x, c.y = x, y
	} else if c.level.isFloor(x, c.y) {
		c.x = x
	} else if c.level.isFloor(c.x, y) {
		c.y = y
	} else {
		c.nextAction = 0
		return
	}

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
		return 150
	default:
		return 0
	}
}
