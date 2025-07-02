package integration

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/blocky/compiler/test"
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Fprintln(os.Stdout, "skipping integration tests in short mode.")
		return
	}

	if !test.ContainerRuntimeAvailable() {
		fmt.Fprintln(os.Stderr, "container runtime not available.")
		os.Exit(1)
	}
	m.Run()
}
