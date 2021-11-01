//go:build js && wasm
// +build js,wasm

package main

func parseFlags(g *game) {
	g.disableEsc = true
}
