package main

import (
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// BatSpriteSheet represents a collection of sprite images.
type BatSpriteSheet struct {
	Frame1 *ebiten.Image
	Frame2 *ebiten.Image
	Frame3 *ebiten.Image
	Frame4 *ebiten.Image
	Frame5 *ebiten.Image
	Frame6 *ebiten.Image
	Frame7 *ebiten.Image
}

// LoadBatSpriteSheet loads the embedded BatSpriteSheet.
func LoadBatSpriteSheet() (*BatSpriteSheet, error) {
	tileSize := 32

	f, err := assetsFS.Open("assets/creeps/bat/fly.png")
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

	// Populate PlayerSpriteSheet.
	s := &BatSpriteSheet{}
	s.Frame1 = spriteAt(0, 0)
	s.Frame2 = spriteAt(1, 0)
	s.Frame3 = spriteAt(2, 0)
	s.Frame4 = spriteAt(3, 0)
	s.Frame5 = spriteAt(4, 0)
	s.Frame6 = spriteAt(5, 0)
	s.Frame7 = spriteAt(6, 0)

	return s, nil
}
