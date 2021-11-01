package main

import (
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

var sandstoneSS *EnvironmentSpriteSheet

// EnvironmentSpriteSheet represents a collection of sprite images.
type EnvironmentSpriteSheet struct {
	FloorA            *ebiten.Image
	FloorB            *ebiten.Image
	FloorC            *ebiten.Image
	WallTop           *ebiten.Image
	WallBottom        *ebiten.Image
	WallBottomLeft    *ebiten.Image
	WallBottomRight   *ebiten.Image
	WallLeft          *ebiten.Image
	WallRight         *ebiten.Image
	WallTopLeft       *ebiten.Image
	WallTopRight      *ebiten.Image
	WallPillar        *ebiten.Image
	TorchTop1         *ebiten.Image
	TorchTop2         *ebiten.Image
	TorchTop3         *ebiten.Image
	TorchTop4         *ebiten.Image
	TorchTop5         *ebiten.Image
	TorchTop6         *ebiten.Image
	TorchTop7         *ebiten.Image
	TorchTop8         *ebiten.Image
	TorchTop9         *ebiten.Image
	TorchMulti        *ebiten.Image
	TopDoorClosedL    *ebiten.Image
	TopDoorClosedR    *ebiten.Image
	TopDoorOpenTL     *ebiten.Image
	TopDoorOpenTR     *ebiten.Image
	TopDoorOpenBL     *ebiten.Image
	TopDoorOpenBR     *ebiten.Image
	BottomDoorClosedL *ebiten.Image
	BottomDoorClosedR *ebiten.Image
	BottomDoorOpenTL  *ebiten.Image
	BottomDoorOpenTR  *ebiten.Image
	BottomDoorOpenBL  *ebiten.Image
	BottomDoorOpenBR  *ebiten.Image
}

// LoadEnvironmentSpriteSheet loads the embedded EnvironmentSpriteSheet.
func LoadEnvironmentSpriteSheet() (*EnvironmentSpriteSheet, error) {
	tileSize := 32

	// Populate EnvironmentSpriteSheet.
	s := &EnvironmentSpriteSheet{}

	// Dungeon sprites
	dungeonFile, err := assetsFS.Open("assets/sandstone-dungeon/Tiles-Sandstone-Dungeons.png")
	if err != nil {
		return nil, err
	}
	defer dungeonFile.Close()
	dungeonImg, _, err := image.Decode(dungeonFile)
	if err != nil {
		return nil, err
	}
	dungeonSheet := ebiten.NewImageFromImage(dungeonImg)
	// dungeonSpriteAt returns a sprite at the provided coordinates.
	dungeonSpriteAt := func(x, y int) *ebiten.Image {
		return dungeonSheet.SubImage(image.Rect(x*tileSize, (y+1)*tileSize, (x+1)*tileSize, y*tileSize)).(*ebiten.Image)
	}
	s.FloorA = dungeonSpriteAt(3, 1)
	s.FloorB = dungeonSpriteAt(2, 0)
	s.FloorC = dungeonSpriteAt(2, 1)
	s.WallTop = dungeonSpriteAt(1, 2)
	s.WallBottom = dungeonSpriteAt(1, 6)
	s.WallBottomLeft = dungeonSpriteAt(2, 6)
	s.WallBottomRight = dungeonSpriteAt(0, 6)
	s.WallLeft = dungeonSpriteAt(2, 2)
	s.WallRight = dungeonSpriteAt(0, 2)
	s.WallTopLeft = dungeonSpriteAt(2, 3)
	s.WallTopRight = dungeonSpriteAt(0, 3)
	s.WallPillar = dungeonSpriteAt(8, 4)

	// Door sprites
	doorFile, err := assetsFS.Open("assets/sandstone-dungeon/Tiles-Door-packs.png")
	if err != nil {
		return nil, err
	}
	defer doorFile.Close()
	doorImg, _, err := image.Decode(doorFile)
	if err != nil {
		return nil, err
	}
	doorSheet := ebiten.NewImageFromImage(doorImg)
	// doorSpriteAt returns a sprite at the provided coordinates.
	doorSpriteAt := func(x, y int) *ebiten.Image {
		return doorSheet.SubImage(image.Rect(x*tileSize, (y+1)*tileSize, (x+1)*tileSize, y*tileSize)).(*ebiten.Image)
	}
	s.TopDoorClosedL = doorSpriteAt(5, 0)
	s.TopDoorClosedR = doorSpriteAt(6, 0)
	s.TopDoorOpenTL = doorSpriteAt(5, 3)
	s.TopDoorOpenTR = doorSpriteAt(6, 3)
	s.TopDoorOpenBL = doorSpriteAt(5, 4)
	s.TopDoorOpenBR = doorSpriteAt(6, 4)
	s.BottomDoorClosedL = dungeonSpriteAt(4, 0)
	s.BottomDoorClosedR = dungeonSpriteAt(5, 0)
	s.BottomDoorOpenTL = doorSpriteAt(5, 3)
	s.BottomDoorOpenTR = doorSpriteAt(6, 3)
	s.BottomDoorOpenBL = doorSpriteAt(5, 4)
	s.BottomDoorOpenBR = doorSpriteAt(6, 4)

	// Prop sprites
	propFile, err := assetsFS.Open("assets/sandstone-dungeon/Tiles-Props-pack.png")
	if err != nil {
		return nil, err
	}
	defer propFile.Close()
	propImg, _, err := image.Decode(propFile)
	if err != nil {
		return nil, err
	}
	propSheet := ebiten.NewImageFromImage(propImg)
	// dungeonSpriteAt returns a sprite at the provided coordinates.
	propSpriteAt := func(x, y int) *ebiten.Image {
		return propSheet.SubImage(image.Rect(x*tileSize, (y+1)*tileSize, (x+1)*tileSize, y*tileSize)).(*ebiten.Image)
	}
	s.TorchTop1 = propSpriteAt(0, 2)
	s.TorchTop2 = propSpriteAt(1, 2)
	s.TorchTop3 = propSpriteAt(2, 2)
	s.TorchTop4 = propSpriteAt(3, 2)
	s.TorchTop5 = propSpriteAt(4, 2)
	s.TorchTop6 = propSpriteAt(5, 2)
	s.TorchTop7 = propSpriteAt(6, 2)
	s.TorchTop8 = propSpriteAt(7, 2)
	s.TorchTop9 = propSpriteAt(8, 2)
	s.TorchMulti = propSpriteAt(2, 4)

	return s, nil
}
