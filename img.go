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
	ImageGarlic:      "assets/items/garlic.png",
	ImageHolyWater:   "assets/items/holywater.png",
	ImageHeart:       "assets/ui/heart.png",
	ImageUzi:         "assets/weapons/uzi.png",
	ImageBullet:      "assets/weapons/bullet.png",
	ImageMuzzleFlash: "assets/weapons/flash.png",
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
