package integration_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var cliCmd = flag.String(
	"cli",
	"go run ../../cmd/bky-c/...",
	"command to run tested CLI",
)

func containerRuntimeAvailable() bool {
	return exec.Command("docker", "version").Run() == nil
}

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Fprintln(os.Stdout, "skipping integration tests in short mode.")
		return
	}

	if !containerRuntimeAvailable() {
		fmt.Fprintln(os.Stderr, "container runtime not available.")
		os.Exit(1)
	}
	m.Run()
}
