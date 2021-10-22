//go:build debug

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		log.Fatal(http.ListenAndServe("localhost:8080", nil))
	}()
}
