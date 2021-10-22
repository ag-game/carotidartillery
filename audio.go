package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	SoundGunshot = iota
	SoundVampireDie1
	SoundVampireDie2
	SoundBat
	SoundPlayerHurt
	SoundPlayerDie
	SoundMunch
	SoundGib
)

var soundMap = map[int]string{
	SoundGunshot:     "assets/audio/gunshot.wav",
	SoundVampireDie1: "assets/audio/vampiredie1.wav",
	SoundVampireDie2: "assets/audio/vampiredie2.wav",
	SoundBat:         "assets/audio/bat.wav",
	SoundPlayerHurt:  "assets/audio/playerhurt.wav",
	SoundPlayerDie:   "assets/audio/playerdie.wav",
	SoundMunch:       "assets/audio/munch.wav",
}
var soundAtlas [][]*audio.Player

var nextSound = make([]int, len(soundMap))

func loadWav(context *audio.Context, p string) (*audio.Player, error) {
	f, err := assetsFS.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stream, err := wav.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, err
	}

	return context.NewPlayer(stream)
}

func loadStream(ctx *audio.Context, p string) (*audio.Player, error) {
	stream, err := loadWav(ctx, p)
	if err != nil {
		return nil, err
	}

	// Workaround to prevent delays when playing for the first time.
	stream.SetVolume(0)
	stream.Play()
	stream.Pause()
	stream.Rewind()

	return stream, nil
}

func loadSoundAtlas(ctx *audio.Context) [][]*audio.Player {
	atlas := make([][]*audio.Player, len(soundMap))
	var err error
	for soundID, soundPath := range soundMap {
		atlas[soundID] = make([]*audio.Player, 4)
		for i := 0; i < 4; i++ {
			atlas[soundID][i], err = loadStream(ctx, soundPath)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return atlas
}
