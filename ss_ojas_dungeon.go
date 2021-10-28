package main

import (
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

var ojasDungeonSS *OjasDungeonSpriteSheet

// OjasDungeonSpriteSheet represents a collection of sprite images.
type OjasDungeonSpriteSheet struct {
	Grass11 *ebiten.Image
	Grass12 *ebiten.Image
	Grass13 *ebiten.Image
	Grass14 *ebiten.Image
	Grass15 *ebiten.Image
	Grass16 *ebiten.Image
	Grass21 *ebiten.Image
	Grass31 *ebiten.Image
	Grass41 *ebiten.Image
	Grass42 *ebiten.Image
	Grass51 *ebiten.Image
	Grass61 *ebiten.Image
	Grass71 *ebiten.Image
	Grass81 *ebiten.Image
	Grass82 *ebiten.Image
	Grass91 *ebiten.Image
	Wall1   *ebiten.Image
	Vent1   *ebiten.Image
	Door11  *ebiten.Image
	Door12  *ebiten.Image
}

// LoadOjasDungeonSpriteSheet loads the embedded PlayerSpriteSheet.
func LoadOjasDungeonSpriteSheet() (*OjasDungeonSpriteSheet, error) {
	tileSize := 32

	f, err := assetsFS.Open("assets/ojas-dungeon/dungeon-tileset-1.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	sheet := ebiten.NewImageFromImage(img)

	// spriteAt returns a sprite at the provided coordinates.
	spriteAt := func(x, y int) *ebiten.Image {
		return sheet.SubImage(image.Rect(x*tileSize, (y+1)*tileSize, (x+1)*tileSize, y*tileSize)).(*ebiten.Image)
	}

	// Populate spritesheet.
	s := &OjasDungeonSpriteSheet{}
	s.Grass11 = spriteAt(7, 7)
	s.Grass12 = spriteAt(8, 7)
	s.Grass13 = spriteAt(9, 7)
	s.Grass14 = spriteAt(10, 7)
	s.Grass15 = spriteAt(11, 7)
	s.Grass16 = spriteAt(12, 7)
	s.Grass21 = spriteAt(10, 8)
	s.Grass31 = spriteAt(10, 9)
	s.Grass41 = spriteAt(15, 9)
	s.Grass42 = spriteAt(10, 10)
	s.Grass51 = spriteAt(10, 11)
	s.Grass61 = spriteAt(10, 12)
	s.Grass71 = spriteAt(10, 13)
	s.Grass81 = spriteAt(14, 11)
	s.Grass82 = spriteAt(10, 14)
	s.Grass91 = spriteAt(10, 15)
	s.Wall1 = spriteAt(7, 5)
	s.Vent1 = spriteAt(9, 13)
	s.Door11 = spriteAt(3, 6)
	s.Door12 = spriteAt(3, 7)

	return s, nil
}
