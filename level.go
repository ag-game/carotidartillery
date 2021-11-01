package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/Meshiest/go-dungeon/dungeon"
)

const dungeonScale = 4

// Level represents a game level.
type Level struct {
	num int

	w, h int

	tiles    [][]*Tile // (Y,X) array of tiles
	tileSize int

	topWalls   [][]*Tile
	sideWalls  [][]*Tile
	otherWalls [][]*Tile

	items []*gameItem

	creeps     []*gameCreep
	liveCreeps int

	player *gamePlayer

	torches []*gameCreep

	enterX, enterY int
	exitX, exitY   int

	exitOpenTime time.Time

	requiredSouls int
}

// NewLevel returns a new randomly generated Level.
func NewLevel(levelNum int, p *gamePlayer) (*Level, error) {
	levelSize := 100
	if levelNum == 2 {
		levelSize = 108
	} else if levelNum == 3 {
		levelSize = 116
	} else if levelSize == 4 {
		levelSize = 256
	}
	// Note: Level size must be divisible by the dungeon scale (4).
	l := &Level{
		num:      levelNum,
		w:        levelSize,
		h:        levelSize,
		tileSize: 32,
		player:   p,
	}

	l.requiredSouls = 33
	if levelNum == 2 {
		l.requiredSouls = 66
	} else if levelNum == 3 {
		l.requiredSouls = 99
	}

	var err error
	sandstoneSS, err = LoadEnvironmentSpriteSheet()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	rooms := 13
	if levelNum == 2 {
		rooms = 26
	} else if levelNum == 3 {
		rooms = 33
	}
	d := dungeon.NewDungeon(l.w/dungeonScale, rooms)
	dungeonFloor := 1
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			if y < l.h-1 && d.Grid[x/dungeonScale][y/dungeonScale] == dungeonFloor {
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

	l.topWalls = make([][]*Tile, l.h)
	l.sideWalls = make([][]*Tile, l.h)
	l.otherWalls = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.topWalls[y] = make([]*Tile, l.w)
		l.sideWalls[y] = make([]*Tile, l.w)
		l.otherWalls[y] = make([]*Tile, l.w)
	}

	// Entrance and exit candidates.
	var topWalls [][2]int
	var bottomWalls [][2]int

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
						c := newCreep(TypeTorch, l, l.player)
						c.x, c.y = float64(nx), float64(ny)
						l.creeps = append(l.creeps, c)
						l.torches = append(l.torches, c)
					} else {
						neighbor.AddSprite(sandstoneSS.WallTop)

						farRight := floorTile(nx+2, ny)
						farBottomRight := floorTile(nx+2, ny+1)
						if bottomRight && farBottomRight && !right && !farRight && y > 2 {
							topWalls = append(topWalls, [2]int{nx, ny})
						}
					}

					l.topWalls[ny][nx] = neighbor
				case spriteLeft:
					if spriteBottom {
						neighbor.AddSprite(sandstoneSS.WallBottom)
					}
					neighbor.AddSprite(sandstoneSS.WallLeft)

					l.sideWalls[ny][nx] = neighbor
					l.otherWalls[ny][nx] = neighbor
				case spriteRight:
					if spriteBottom {
						neighbor.AddSprite(sandstoneSS.WallBottom)
					}
					neighbor.AddSprite(sandstoneSS.WallRight)

					l.sideWalls[ny][nx] = neighbor
					l.otherWalls[ny][nx] = neighbor
				case spriteBottomLeft:
					neighbor.AddSprite(sandstoneSS.WallBottomLeft)

					l.sideWalls[ny][nx] = neighbor
					l.otherWalls[ny][nx] = neighbor
				case spriteBottomRight:
					neighbor.AddSprite(sandstoneSS.WallBottomRight)

					l.sideWalls[ny][nx] = neighbor
					l.otherWalls[ny][nx] = neighbor
				case spriteBottom:
					neighbor.AddSprite(sandstoneSS.WallBottom)

					l.otherWalls[ny][nx] = neighbor

					farRight := floorTile(nx+2, ny)
					farTopRight := floorTile(nx+2, ny-1)
					if topRight && farTopRight && !right && !farRight && ny < l.h-3 {
						bottomWalls = append(bottomWalls, [2]int{nx, ny})
					}
				}
			}
		}
	}

	for {
		entrance := bottomWalls[rand.Intn(len(bottomWalls))]
		l.enterX, l.enterY = entrance[0], entrance[1]

		exit := topWalls[rand.Intn(len(topWalls))]
		l.exitX, l.exitY = exit[0], exit[1]

		dx, dy := deltaXY(float64(l.enterX), float64(l.enterY), float64(l.exitX), float64(l.exitY))
		if dy >= 8 || dx >= 6 {
			break
		}
	}

	fadeA := 0.15
	fadeB := 0.1

	// Add entrance.
	if levelNum > 1 {
		t := l.Tile(l.enterX, l.enterY)
		t.sprites = nil
		t.AddSprite(sandstoneSS.FloorA)
		t.AddSprite(sandstoneSS.BottomDoorClosedL)
		t.AddSprite(sandstoneSS.WallLeft)

		t = l.Tile(l.enterX+1, l.enterY)
		t.sprites = nil
		t.AddSprite(sandstoneSS.FloorA)
		t.AddSprite(sandstoneSS.BottomDoorClosedR)
		t.AddSprite(sandstoneSS.WallRight)

		// Add fading entrance hall.
		for i := 1; i < 3; i++ {
			colorScale := fadeA
			if i == 2 {
				colorScale = fadeB
			}

			t = l.Tile(l.enterX, l.enterY+i)
			if t != nil {
				t.AddSprite(sandstoneSS.FloorA)
				t.AddSprite(sandstoneSS.WallLeft)
				t.forceColorScale = colorScale
			}

			t = l.Tile(l.enterX+1, l.enterY+i)
			if t != nil {
				t.AddSprite(sandstoneSS.FloorA)
				t.AddSprite(sandstoneSS.WallRight)
				t.forceColorScale = colorScale
			}
		}
	}

	// Add exit.
	t := l.Tile(l.exitX, l.exitY)
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorClosedL)

	t = l.Tile(l.exitX+1, l.exitY)
	t.sprites = nil
	t.AddSprite(sandstoneSS.FloorA)
	t.AddSprite(sandstoneSS.TopDoorClosedR)

	// Add fading exit hall.
	for i := 1; i < 3; i++ {
		colorScale := fadeA
		if i == 2 {
			colorScale = fadeB
		}

		t = l.Tile(l.exitX, l.exitY-i)
		if t != nil {
			t.AddSprite(sandstoneSS.FloorA)
			t.AddSprite(sandstoneSS.WallLeft)
			t.forceColorScale = colorScale
		}

		t = l.Tile(l.exitX+1, l.exitY-i)
		if t != nil {
			t.AddSprite(sandstoneSS.FloorA)
			t.AddSprite(sandstoneSS.WallRight)
			t.forceColorScale = colorScale
		}
	}

	// TODO make it more obvious players should enter it (arrow on first level?)

	// TODO two frame sprite arrow animation

	// TODO special door for final exit

	l.bakeLightmap()

	return l, nil
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

func (l *Level) isFloor(x float64, y float64) bool {
	t := l.Tile(int(math.Floor(x+.5)), int(math.Floor(y+.5)))
	if t == nil {
		return false
	}
	if !t.floor {
		return false
	}
	return true
}

func (l *Level) newSpawnLocation() (float64, float64) {
SPAWNLOCATION:
	for {
		x := float64(1 + rand.Intn(l.w-2))
		y := float64(1 + rand.Intn(l.h-2))

		if !l.isFloor(x, y) {
			continue
		}

		// Too close to player.
		playerSafeSpace := 11.0
		dx, dy := deltaXY(x, y, l.player.x, l.player.y)
		if dx <= playerSafeSpace && dy <= playerSafeSpace {
			continue
		}

		// Too close to entrance.
		exitSafeSpace := 9.0
		dx, dy = deltaXY(x, y, float64(l.enterX), float64(l.enterY))
		if dx <= exitSafeSpace && dy <= exitSafeSpace {
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

func (l *Level) bakeLightmap() {
	for x := 0; x < l.w; x++ {
		for y := 0; y < l.h; y++ {
			t := l.tiles[y][x]
			v := t.forceColorScale
			if v == 0 {
				for _, torch := range l.torches {
					if torch.health == 0 {
						continue
					}
					torchV := colorScaleValue(float64(x), float64(y), torch.x, torch.y)
					v += torchV
				}
			}
			t.colorScale = v
		}
	}
}

func (l *Level) bakePartialLightmap(lx, ly int) {
	radius := 16
	for x := lx - radius; x < lx+radius; x++ {
		for y := ly - radius; y < ly+radius; y++ {
			t := l.Tile(x, y)
			if t == nil {
				continue
			}
			v := 0.0
			for _, torch := range l.torches {
				if torch.health == 0 {
					continue
				}
				torchV := colorScaleValue(float64(x), float64(y), torch.x, torch.y)
				v += torchV
			}
			t.colorScale = v
		}
	}
}

func (l *Level) addCreep(creepType int) *gameCreep {
	c := newCreep(creepType, l, l.player)
	l.creeps = append(l.creeps, c)
	return c
}

func angle(x1, y1, x2, y2 float64) float64 {
	return math.Atan2(y1-y2, x1-x2)
}

func colorScaleValue(x, y, bx, by float64) float64 {
	dx, dy := deltaXY(x, y, bx, by)
	sD := 7 / (dx + dy)
	if sD > 1 {
		sD = 1
	}
	sDB := sD
	if dx > 4 {
		sDB *= 0.6 / (dx / 4)
	}
	if dy > 4 {
		sDB *= 0.6 / (dy / 4)
	}
	sD = sD * 2 * sDB
	if sD > 1 {
		sD = 1
	}
	return sD
}
