package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func newWinLevel(p *gamePlayer) *Level {
	l := &Level{
		w:        256,
		h:        256,
		tileSize: 32,
		player:   p,
	}

	startX, startY := 108, 108

	grid := make([][][]*ebiten.Image, l.w)
	for x := 0; x < l.w; x++ {
		grid[x] = make([][]*ebiten.Image, l.h)
	}

	// Add ground.
	var lastBones int
	bonesDistance := 7
	for x := 0; x < l.w; x++ {
		excludeBones := x-lastBones < bonesDistance

		r := rand.Intn(33)
		switch r {
		case 0:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass11,
			}
		case 1:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass12,
			}
		case 2:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass13,
			}
		case 3:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass14,
			}
		case 4:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass15,
			}
		case 5:
			grid[x][startY] = []*ebiten.Image{
				ojasDungeonSS.Grass16,
			}
		}
		grid[x][startY+1] = []*ebiten.Image{
			ojasDungeonSS.Grass21,
		}
		grid[x][startY+2] = []*ebiten.Image{
			ojasDungeonSS.Grass31,
		}
		if rand.Intn(33) != 0 || excludeBones {
			grid[x][startY+3] = []*ebiten.Image{
				ojasDungeonSS.Grass41,
			}
		} else {
			grid[x][startY+3] = []*ebiten.Image{
				ojasDungeonSS.Grass42,
			}
		}
		grid[x][startY+4] = []*ebiten.Image{
			ojasDungeonSS.Grass51,
		}
		grid[x][startY+5] = []*ebiten.Image{
			ojasDungeonSS.Grass61,
		}
		grid[x][startY+6] = []*ebiten.Image{
			ojasDungeonSS.Grass71,
		}
		if rand.Intn(33) != 0 || excludeBones {
			grid[x][startY+7] = []*ebiten.Image{
				ojasDungeonSS.Grass81,
			}
		} else {
			grid[x][startY+7] = []*ebiten.Image{
				ojasDungeonSS.Grass82,
			}
		}
		grid[x][startY+8] = []*ebiten.Image{
			ojasDungeonSS.Grass91,
		}
	}

	// Add dungeon.
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			grid[startX-x-1][startY-y] = []*ebiten.Image{
				ojasDungeonSS.Wall1,
			}

			if y == 0 && x%8 == 0 {
				grid[startX-x-1][startY-y] = append(grid[startX-x-1][startY-y], ojasDungeonSS.Vent1)
			}
		}
	}

	grid[startX-1][startY-1] = append(grid[startX-1][startY-1], ojasDungeonSS.Door11)
	grid[startX-1][startY] = append(grid[startX-1][startY], ojasDungeonSS.Door12)

	// Add sprites to tiles.
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			for _, sprite := range grid[x][y] {
				t.AddSprite(sprite)
			}
			l.tiles[y][x] = t
		}
	}

	l.bakeLightmap()

	doorX := float64(startX) - 0.4

	p.angle = 0
	p.x, p.y = doorX, float64(startY)

	go func() {
		// Walk away.
		for i := 0; i < 36; i++ {
			p.x += 0.05
			time.Sleep(time.Second / 144)
		}
		for i := 0; i < 288; i++ {
			p.x += 0.05 * (float64(288-i) / 288)
			time.Sleep(time.Second / 144)
		}

		// Turn around.
		p.angle = math.Pi
		time.Sleep(time.Millisecond * 1750)

		// Throw weapon.
		weaponSprite := newCreep(TypeTorch, l, p)
		weaponSprite.x, weaponSprite.y = p.x, p.y-0.25
		weaponSprite.frames = 1
		weaponSprite.frame = 0
		weaponSprite.sprites = []*ebiten.Image{
			imageAtlas[ImageUzi],
		}

		p.weapon = nil
		l.creeps = append(l.creeps, weaponSprite)

		go func() {
			for i := 0; i < 144*2; i++ {
				if weaponSprite.x < doorX {
					for i, c := range l.creeps {
						if c == weaponSprite {
							l.creeps = append(l.creeps[:i], l.creeps[i+1:]...)
						}
					}
					return
				}

				weaponSprite.x -= 0.05
				if i < 100 {
					weaponSprite.y -= 0.005 * (float64(144-i) / 144)
				} else {
					weaponSprite.y += 0.01 * (float64(288-i) / 288)
				}
				weaponSprite.angle -= .1
				time.Sleep(time.Second / 144)
			}
		}()

		time.Sleep(time.Second / 2)

		// Throw torch.
		torchSprite := newCreep(TypeTorch, l, p)
		torchSprite.x, torchSprite.y = p.x, p.y-0.25
		torchSprite.frames = 1
		torchSprite.frame = 0
		torchSprite.sprites = []*ebiten.Image{
			sandstoneSS.TorchMulti,
		}

		p.hasTorch = false
		l.creeps = append(l.creeps, torchSprite)

		go func() {
			for i := 0; i < 144*3; i++ {
				if torchSprite.x < doorX {
					for i, c := range l.creeps {
						if c == torchSprite {
							l.creeps = append(l.creeps[:i], l.creeps[i+1:]...)
						}
					}
				}

				torchSprite.x -= 0.05
				if i < 100 {
					torchSprite.y -= 0.005 * (float64(144-i) / 144)
				} else {
					torchSprite.y += 0.01 * (float64(288-i) / 288)
				}

				torchSprite.angle -= .1
				time.Sleep(time.Second / 144)
			}
		}()

		// Walk away.
		time.Sleep(time.Second)

		p.angle = 0
		for i := 0; i < 144; i++ {
			p.x += 0.05 * (float64(i) / 144)
			time.Sleep(time.Second / 144)
		}
		for i := 0; i < 144*15; i++ {
			if p.health > 0 {
				// Game has restarted.
				return
			}
			p.x += 0.05
			time.Sleep(time.Second / 144)
		}
	}()

	return l
}
