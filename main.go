package main

import (
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ebiten.SetWindowTitle("Carotid Artillery")
	ebiten.SetWindowResizable(true)
	ebiten.SetFullscreen(true)
	ebiten.SetMaxTPS(144)
	ebiten.SetRunnableOnUnfocused(true) // Note - this currently does nothing in ebiten
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)

	g, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}

	err = g.reset()
	if err != nil {
		panic(err)
	}

	parseFlags(g)
	if !g.debugMode {
		g.gameStartTime = time.Time{}
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM)
	go func() {
		<-sigc

		g.exit()
	}()

	err = ebiten.RunGame(g)
	if err != nil {
		log.Fatal(err)
	}
}
