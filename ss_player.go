package main

import (
	"embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

var ojasSS *PlayerSpriteSheet

// PlayerSpriteSheet represents a collection of sprite images.
type PlayerSpriteSheet struct {
	Frame1 *ebiten.Image
	Frame2 *ebiten.Image
}

//go:embed assets
var assetsFS embed.FS

// LoadPlayerSpriteSheet loads the embedded PlayerSpriteSheet.
func LoadPlayerSpriteSheet() (*PlayerSpriteSheet, error) {
	tileSize := 32

	f, err := assetsFS.Open("assets/ojas-dungeon/character_run.png")
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
	s := &PlayerSpriteSheet{}
	s.Frame1 = spriteAt(0, 0)
	s.Frame2 = spriteAt(0, 1)

	return s, nil
}
