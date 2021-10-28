//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"flag"
)

func parseFlags(g *game) {
	flag.BoolVar(&g.godMode, "god", false, "Enable God mode")
	flag.BoolVar(&g.noclipMode, "noclip", false, "Enable noclip mode")
	flag.BoolVar(&g.fullBrightMode, "fullbright", false, "Enable fullbright mode")
	flag.BoolVar(&g.debugMode, "debug", false, "Enable debug mode")
	flag.BoolVar(&g.muteAudio, "mute", false, "Mute audio")
	flag.Parse()
}
