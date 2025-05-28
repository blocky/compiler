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
	"go run ../../cmd/cli/...",
	"command to run tested CLI",
)

func containerRuntimeAvailable() bool {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
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
