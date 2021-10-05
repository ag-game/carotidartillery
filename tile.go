package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Tile represents a space with an x,y coordinate within a Level. Any number of
// sprites may be added to a Tile.
type Tile struct {
	sprites []*ebiten.Image
}

// AddSprite adds a sprite to the Tile.
func (t *Tile) AddSprite(s *ebiten.Image) {
	t.sprites = append(t.sprites, s)
}

// ClearSprites removes all sprites from the Tile.
func (t *Tile) ClearSprites() {
	for i := range t.sprites {
		t.sprites[i] = nil
	}
	t.sprites = t.sprites[:0]
}

// Draw draws the Tile on the screen using the provided options.
func (t *Tile) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for _, s := range t.sprites {
		screen.DrawImage(s, options)
	}
}
