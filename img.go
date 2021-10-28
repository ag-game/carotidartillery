package main

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ImageVampire1 = iota
	ImageVampire2
	ImageVampire3
	ImageGhost1
	ImageGhost1R
	ImageGhost2
	ImageGhost2R
	ImageGarlic
	ImageHolyWater
	ImageHeart
	ImageUzi
	ImageBullet
	ImageMuzzleFlash
)

var imageMap = map[int]string{
	ImageVampire1:    "assets/creeps/vampire/vampire1.png",
	ImageVampire2:    "assets/creeps/vampire/vampire2.png",
	ImageVampire3:    "assets/creeps/vampire/vampire3.png",
	ImageGhost1:      "assets/creeps/ghost/ghost1.png",
	ImageGhost1R:     "assets/creeps/ghost/ghost1r.png",
	ImageGhost2:      "assets/creeps/ghost/ghost2.png",
	ImageGhost2R:     "assets/creeps/ghost/ghost2r.png",
	ImageGarlic:      "assets/items/garlic.png",
	ImageHolyWater:   "assets/items/holy-water.png",
	ImageHeart:       "assets/ui/heart.png",
	ImageUzi:         "assets/weapons/uzi.png",
	ImageBullet:      "assets/weapons/bullet.png",
	ImageMuzzleFlash: "assets/weapons/muzzle-flash.png",
}

var imageAtlas = loadAtlas()

func loadImage(p string) (*ebiten.Image, error) {
	f, err := assetsFS.Open(p)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func loadAtlas() []*ebiten.Image {
	atlas := make([]*ebiten.Image, len(imageMap))
	var err error
	for imgID, imgPath := range imageMap {
		atlas[imgID], err = loadImage(imgPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	return atlas
}
