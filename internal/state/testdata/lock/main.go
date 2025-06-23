package main

import (
	"flag"
	"log"

	"github.com/blocky/compiler/internal/state"
)

func main() {
	dir := flag.String("path", "", "path to lock")
	flag.Parse()
	_, err := state.Lock(*dir)
	if err != nil {
		log.Fatal(err)
	}
}
