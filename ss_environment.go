package main

import (
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// EnvironmentSpriteSheet represents a collection of sprite images.
type EnvironmentSpriteSheet struct {
	FloorA          *ebiten.Image
	FloorB          *ebiten.Image
	FloorC          *ebiten.Image
	WallTop         *ebiten.Image
	WallBottom      *ebiten.Image
	WallBottomLeft  *ebiten.Image
	WallBottomRight *ebiten.Image
	WallLeft        *ebiten.Image
	WallRight       *ebiten.Image
	WallTopLeft     *ebiten.Image
	WallTopRight    *ebiten.Image
	WallPillar      *ebiten.Image
}

// LoadEnvironmentSpriteSheet loads the embedded EnvironmentSpriteSheet.
func LoadEnvironmentSpriteSheet() (*EnvironmentSpriteSheet, error) {
	tileSize := 32

	f, err := assetsFS.Open("assets/sandstone-dungeon/Tiles-SandstoneDungeons.png")
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

	// Populate EnvironmentSpriteSheet.
	s := &EnvironmentSpriteSheet{}
	s.FloorA = spriteAt(3, 1)
	s.FloorB = spriteAt(2, 0)
	s.FloorC = spriteAt(2, 1)
	s.WallTop = spriteAt(1, 2)
	s.WallBottom = spriteAt(1, 6)
	s.WallBottomLeft = spriteAt(2, 6)
	s.WallBottomRight = spriteAt(0, 6)
	s.WallLeft = spriteAt(2, 2)
	s.WallRight = spriteAt(0, 2)
	s.WallTopLeft = spriteAt(2, 3)
	s.WallTopRight = spriteAt(0, 3)
	s.WallPillar = spriteAt(8, 4)

	return s, nil
}
