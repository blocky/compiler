package main

import (
	"log"

	"github.com/blocky/bkyc/cmd/bky-c/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
