package main

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/Meshiest/go-dungeon/dungeon"
)

const dungeonScale = 4

// Level represents a game level.
type Level struct {
	w, h int

	tiles    [][]*Tile // (Y,X) array of tiles
	tileSize int

	items []*gameItem

	creeps     []*gameCreep
	liveCreeps int

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

func (l *Level) isFloor(x float64, y float64, exact bool) bool {
	offsetA := .25
	offsetB := .75
	if exact {
		offsetA = 0
		offsetB = 0
	}

	t := l.Tile(int(x+offsetA), int(y+offsetA))
	if t == nil {
		return false
	}
	if !t.floor {
		return false
	}

	t = l.Tile(int(x+offsetB), int(y+offsetB))
	if t == nil {
		return false
	}
	return t.floor
}

func (l *Level) newSpawnLocation() (float64, float64) {
SPAWNLOCATION:
	for {
		x := float64(1 + rand.Intn(l.w-2))
		y := float64(1 + rand.Intn(l.h-2))

		if !l.isFloor(x, y, false) {
			continue
		}

		// Too close to player.
		playerSafeSpace := 18.0
		dx, dy := deltaXY(x, y, l.player.x, l.player.y)
		if dx <= playerSafeSpace && dy <= playerSafeSpace {
			continue
		}

		// Too close to garlic or holy water.
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
	// Create a 216x216 Level.
	l := &Level{
		w:        216,
		h:        216,
		tileSize: 32,
	}

	sandstoneSS, err := LoadEnvironmentSpriteSheet()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	dungeon := dungeon.NewDungeon(l.w/dungeonScale, 13)
	dungeonFloor := 1
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			if y < l.h-1 && dungeon.Grid[x/dungeonScale][y/dungeonScale] == dungeonFloor {
				if rand.Intn(13) == 0 {
					t.AddSprite(sandstoneSS.FloorC)
				} else {
					t.AddSprite(sandstoneSS.FloorA)
				}
				t.floor = true
			}
			l.tiles[y][x] = t
		}
	}

	neighbors := func(x, y int) [][2]int {
		return [][2]int{
			{x - 1, y - 1},
			{x, y - 1},
			{x + 1, y - 1},
			{x + 1, y},
			{x + 1, y + 1},
			{x, y + 1},
			{x - 1, y + 1},
			{x - 1, y},
		}
	}

	floorTile := func(x, y int) bool {
		t := l.Tile(x, y)
		if t == nil {
			return false
		}
		return t.floor
	}

	// Add walls.
	for x := 0; x < l.w; x++ {
		for y := 0; y < l.h; y++ {
			t := l.Tile(x, y)
			if t == nil {
				continue
			}
			if !t.floor {
				continue
			}
			for _, n := range neighbors(x, y) {
				nx, ny := n[0], n[1]
				neighbor := l.Tile(nx, ny)
				if neighbor == nil || neighbor.floor || neighbor.wall {
					continue
				}
				neighbor.wall = true

				// From perspective of neighbor tile.
				bottom := floorTile(nx, ny+1)
				top := floorTile(nx, ny-1)
				right := floorTile(nx+1, ny)
				left := floorTile(nx-1, ny)
				topLeft := floorTile(nx-1, ny-1)
				topRight := floorTile(nx+1, ny-1)
				bottomLeft := floorTile(nx-1, ny+1)
				bottomRight := floorTile(nx+1, ny+1)

				// Determine which wall sprite to belongs here.
				spriteTop := !top && bottom
				spriteLeft := (left || bottomLeft) && !right && !bottomRight && !bottom
				spriteRight := (right || bottomRight) && !left && !bottomLeft && !bottom
				spriteBottomRight := !topLeft && !top && topRight && !bottomLeft && !bottom && !bottomRight
				spriteBottomLeft := topLeft && !top && !topRight && !bottomLeft && !bottom && !bottomRight
				spriteBottom := top && !bottom

				// Add wall sprite.
				switch {
				case spriteTop:
					if !bottomLeft || !bottomRight || left || right {
						neighbor.AddSprite(sandstoneSS.WallPillar)
					} else {
						neighbor.AddSprite(sandstoneSS.WallTop)
					}
				case spriteLeft:
					if spriteBottom {
						neighbor.AddSprite(sandstoneSS.WallBottom)
					}
					neighbor.AddSprite(sandstoneSS.WallLeft)
				case spriteRight:
					if spriteBottom {
						neighbor.AddSprite(sandstoneSS.WallBottom)
					}
					neighbor.AddSprite(sandstoneSS.WallRight)
				case spriteBottomLeft:
					neighbor.AddSprite(sandstoneSS.WallBottomLeft)
				case spriteBottomRight:
					neighbor.AddSprite(sandstoneSS.WallBottomRight)
				case spriteBottom:
					neighbor.AddSprite(sandstoneSS.WallBottom)
				}
			}
		}
	}

	return l, nil
}

func angle(x1, y1, x2, y2 float64) float64 {
	return math.Atan2(y1-y2, x1-x2)
}
