package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Level represents a game level.
type Level struct {
	w, h int

	tiles    [][]*Tile // (Y,X) array of tiles
	tileSize int

	items []*gameItem

	creeps []*gameCreep

	player *gamePlayer
}

// Tile returns the tile at the provided coordinates, or nil.
func (l *Level) Tile(x, y int) *Tile {
	if x >= 0 && y >= 0 && x < l.w && y < l.h {
		return l.tiles[y][x]
	}
	return nil
}

// Size returns the size of the Level.
func (l *Level) Size() (width, height int) {
	return l.w, l.h
}

func (l *Level) Clamp(x, y float64) (float64, float64) {
	if x < 0.3 {
		x = 0.3
	} else if x > float64(l.w)-1.3 {
		x = float64(l.w) - 1.3
	}
	if y < 0.4 {
		y = 0.4
	} else if y > float64(l.h)-1.8 {
		y = float64(l.h) - 1.8
	}
	return x, y
}

func (l *Level) newSpawnLocation() (float64, float64) {
SPAWNLOCATION:
	for {
		x := float64(rand.Intn(108))
		y := float64(rand.Intn(108))

		// Too close to player.
		playerSafeSpace := 7.0
		dx, dy := deltaXY(x, y, l.player.x, l.player.y)
		if dx <= playerSafeSpace && dy <= playerSafeSpace {
			continue
		}

		// Too close to garlic.
		garlicSafeSpace := 2.0
		for _, item := range l.items {
			if item.health == 0 {
				continue
			}

			dx, dy = deltaXY(x, y, item.x, item.y)
			if dx <= garlicSafeSpace && dy <= garlicSafeSpace {
				continue SPAWNLOCATION
			}
		}

		return x, y
	}

}

// NewLevel returns a new randomly generated Level.
func NewLevel() (*Level, error) {
	// Create a 108x108 Level.
	l := &Level{
		w:        108,
		h:        108,
		tileSize: 32,
	}

	sandstoneSS, err := LoadEnvironmentSpriteSheet()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	_ = sandstoneSS

	// Generate a unique permutation each time.
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	// Fill each tile with one or more sprites randomly.
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			t.AddSprite(sandstoneSS.FloorA)

			val := r.Intn(1000)
			switch {
			case x == 0 && y == 0:
				t.AddSprite(sandstoneSS.WallTopLeft)
			case x == l.w-1 && y == 0:
				t.AddSprite(sandstoneSS.WallTopRight)
			case x == 0 && y == l.h-1:
				t.AddSprite(sandstoneSS.WallBottom)
			case x == l.w-1 && y == l.h-1:
				t.AddSprite(sandstoneSS.WallBottom)
			case y == 0:
				if x%(l.w/7) == 1 {
					t.AddSprite(sandstoneSS.WallPillar)
				} else {
					t.AddSprite(sandstoneSS.WallTop)
				}
			case y == l.h-1:
				t.AddSprite(sandstoneSS.WallBottom)
			case x == 0:
				t.AddSprite(sandstoneSS.WallLeft)
			case x == l.w-1:
				t.AddSprite(sandstoneSS.WallRight)
			case val < 275:
				//t.AddSprite(sandstoneSS.FloorB)
			case val < 500:
				t.AddSprite(sandstoneSS.FloorC)
			}
			l.tiles[y][x] = t
		}
	}

	return l, nil
}

func angle(x1, y1, x2, y2 float64) float64 {
	return math.Atan2(y1-y2, x1-x2)
}
