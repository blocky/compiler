package main

import (
	"log"

	"github.com/blocky/bkyc/cmd/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
