package integration_test

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationHelloWorld(t *testing.T) {
	assert.True(t, true)
}

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Fprintln(os.Stdout, "skipping integration tests in short mode.")
		return
	}

	m.Run()
}
