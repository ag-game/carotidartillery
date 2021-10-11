package main

import (
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	SoundGunshot = iota
	SoundVampireDie1
	SoundVampireDie2
	SoundBat
	SoundPlayerHurt
	SoundPlayerDie
	SoundGib
)

func loadMP3(context *audio.Context, p string) (*audio.Player, error) {
	f, err := assetsFS.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stream, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, err
	}

	return context.NewPlayer(stream)
}

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
